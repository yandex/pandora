// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// Stream related functions

package spdy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const INITIAL_FLOW_CONTOL_WINDOW int32 = 64 * 1024
const NORTHBOUND_SLOTS = 5

// NewClientStream starts a new Stream (in the given Session), to be used as a client
func (s *Session) NewClientStream() *Stream {
	// no stream creation after goaway has been recieved
	if !s.goaway_recvd {
		str := &Stream{
			id:                s.nextStreamID(),
			session:           s,
			priority:          4, // FIXME need to implement priorities
			associated_stream: 0, // FIXME for pushes we need to implement it
			control:           make(chan controlFrame),
			data:              make(chan dataFrame),
			response:          make(chan bool),
			eos:               make(chan bool),
			stop_server:       make(chan bool),
			flow_req:          make(chan int32, 1),
			flow_add:          make(chan int32, 1),
			upstream_buffer:   make(chan upstream_data, NORTHBOUND_SLOTS),
		}

		go str.serve()

		go str.northboundBufferSender()

		go str.flowManager(INITIAL_FLOW_CONTOL_WINDOW, str.flow_add, str.flow_req)

		// add the stream to the session

		deadline := time.After(1500 * time.Millisecond)
		select {
		case s.new_stream <- str:
			// done
			return str
		case <-deadline:
			// somehow it was locked
			debug.Printf("Stream #%d: cannot be created. Stream is hung. Resetting it.", str.id)
			s.Close()
			return nil
		}
	} else {
		debug.Println("Cannot create stream after receiving goaway")
		return nil
	}
}

func (s *Session) newServerStream(frame controlFrame) (str *Stream, err error) {
	// no stream creation after goaway has been recieved
	if !s.goaway_recvd {
		str = &Stream{
			id:                frame.streamID(),
			session:           s,
			priority:          4, // FIXME need to implement priorities
			associated_stream: 0, // FIXME for pushes we need to implement it
			control:           make(chan controlFrame),
			data:              make(chan dataFrame),
			response:          make(chan bool),
			eos:               make(chan bool),
			stop_server:       make(chan bool),
			flow_req:          make(chan int32, 1),
			flow_add:          make(chan int32, 1),
		}

		go str.serve()

		go str.flowManager(INITIAL_FLOW_CONTOL_WINDOW, str.flow_add, str.flow_req)

		// send the SYN_STREAM control frame to get it started
		str.control <- frame
		return
	} else {
		return nil, errors.New("Cannot create stream after receiving goaway")
	}

}

// String returns the Stream ID of the Stream
func (s *Stream) String() string {
	return fmt.Sprintf("%d", s.id)
}

// prepare the header of the request in SPDY format
func (s *Stream) prepareRequestHeader(request *http.Request) (err error) {
	url := request.URL
	if url != nil && url.Path == "" {
		url.Path = "/"
	}
	if url == nil || url.Scheme == "" || url.Host == "" || url.Path == "" {
		err = errors.New(fmt.Sprintf("ERROR: Incomplete path provided: scheme=%s host=%s path=%s", url.Scheme, url.Host, url.Path))
		return
	}
	path := url.Path
	if url.RawQuery != "" {
		path += "?" + url.RawQuery
	}
	if url.Fragment != "" {
		path += "#" + url.Fragment
	}

	// normalize host:port
	if !strings.Contains(url.Host, ":") {
		switch url.Scheme {
		case "http":
			url.Host += ":80"
		case "https":
			url.Host += ":443"
		}
	}

	// set all SPDY headers
	request.Header.Set(HEADER_METHOD, request.Method)
	request.Header.Set(HEADER_PATH, path)
	request.Header.Set(HEADER_VERSION, request.Proto)
	request.Header.Set(HEADER_HOST, url.Host)
	request.Header.Set(HEADER_SCHEME, url.Scheme)

	request.Header.Del("Connection")
	request.Header.Del("Host")
	request.Header.Del("Keep-Alive")
	request.Header.Del("Proxy-Connection")
	request.Header.Del("Transfer-Encoding")

	return nil
}

func (s *Stream) prepareRequestBody(request *http.Request) (body []*dataFrame, err error) {

	body = make([]*dataFrame, 0, 1)
	if request.Body == nil {
		return body, nil
	}

	buf := make([]byte, 32*1024)
	n, err := request.Body.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}
	total := n
	for n > 0 {
		data := new(dataFrame)
		data.data = make([]byte, n)
		copy(data.data, buf[:n])
		body = append(body, data)
		n, err = request.Body.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		total += n
	}

	// Half-close the stream.
	if len(body) != 0 {
		request.Header.Set("Content-Length", fmt.Sprint(total))
		body[len(body)-1].flags = FLAG_FIN
	}
	request.Body.Close()

	err = nil // in case it was EOF, which is not an error

	return
}

func (s *Stream) handleRequest(request *http.Request) (err error) {
	err = s.prepareRequestHeader(request)
	if err != nil {
		return
	}

	flags := FLAG_NONE
	if request.Body == nil {
		flags = FLAG_FIN
	}

	body, err := s.prepareRequestBody(request)
	if err != nil {
		return
	}

	if len(body) == 0 {
		flags = FLAG_FIN
	}

	// send the SYN frame to start the stream
	f := frameSynStream{session: s.session, stream: s.id, header: request.Header, flags: flags}
	debug.Println("Sending SYN_STREAM:", f)
	s.session.out <- f

	// send the DATA frames for the body
	for _, frame := range body {
		frame.stream = s.id
		s.session.out <- frame
	}

	// need to return now but the data pieces will be picked up
	// and the eos channel will be notified when done

	return nil
}

// Request makes an http request down the client that gets a client Stream
// started and returning the request in the ResponseWriter
func (s *Stream) Request(request *http.Request, writer http.ResponseWriter) (err error) {

	s.response_writer = writer

	err = s.handleRequest(request)
	if err != nil {
		debug.Println("ERROR in stream.serve/http.Request:", err)
		return
	}

	debug.Printf("Waiting for #%d to end", s.id)

	// the response is finished sending
	<-s.eos

	s.finish_stream()

	return
}

func (s *Stream) finish_stream() {

	defer no_panics()
	deadline := time.After(500 * time.Millisecond)

	select {
	case s.stop_server <- true:
		// done
	case <-deadline:
		// well, somehow it was locked
		debug.Printf("Stream #%d: Request() timed out while stopping", s.id)
	}

}

// Takes a SYN_STREAM control frame and kicks off a stream, calling the handler
func (s *Stream) initiate_stream(frame controlFrame) (err error) {
	debug.Println("Stream server got SYN_STREAM")

	s.id = frame.streamID()

	data := bytes.NewBuffer(frame.data[4:])

	// add the stream to the map of streams in the session
	// note that above the message sending the ID is zero initially!
	s.session.new_stream <- s

	var associated_id uint32
	err = binary.Read(data, binary.BigEndian, &associated_id)
	if err != nil {
		return err
	}
	s.associated_stream = streamID(associated_id & 0x7fffffff)

	var b uint8
	err = binary.Read(data, binary.BigEndian, &b)
	if err != nil {
		return err
	}
	s.priority = (b & (0x7 << 5)) >> 5

	// skip over the "slot" of unused space
	_, err = io.ReadFull(data, make([]byte, 1))
	if err != nil {
		return err
	}

	headers := make(http.Header)
	// debug.Println("header data:", data.Bytes())
	headers, err = s.session.headerReader.decode(data.Bytes())
	if err != nil {
		return err
	}

	headers.Del("Connection")
	headers.Del("Host")
	headers.Del("Keep-Alive")
	headers.Del("Proxy-Connection")
	headers.Del("Transfer-Encoding")

	s.headers = headers

	// build the frame just for printing it
	ss := frameSynStream{
		session:           s.session,
		stream:            s.id,
		priority:          s.priority,
		associated_stream: s.associated_stream,
		header:            headers,
		flags:             frame.flags}

	debug.Println("Processing SYN_STREAM", ss)
	if frame.isFIN() {
		// call the handler

		req := &http.Request{
			Method:     headers.Get(HEADER_METHOD),
			Proto:      headers.Get(HEADER_VERSION),
			Header:     headers,
			RemoteAddr: s.session.conn.RemoteAddr().String(),
		}
		req.URL, _ = url.ParseRequestURI(headers.Get(HEADER_PATH))

		// Clear the headers in the session now that the request has them
		s.headers = make(http.Header)

		go s.requestHandler(req)

	} else {
		var data []byte

		endflag := 0
		for endflag == 0 {
			deadline := time.After(3 * time.Second)
			select {
			case cf, ok := <-s.control:
				if !ok {
					return
				}
				switch cf.kind {
				case FRAME_SYN_STREAM:
					err = errors.New("Multiple Syn Streams sent to single Stream")
					return
				case FRAME_SYN_REPLY:
					err = s.handleSynReply(cf)
				case FRAME_RST_STREAM:
					err = s.handleRstStream(cf)
					return
				case FRAME_WINDOW_UPDATE:
					s.handleWindowUpdate(cf)
				default:
					panic("TODO: unhandled type of frame received in stream.serve()")
				}
			case df, ok := <-s.data:
				//collecting data
				if !ok {
					debug.Println("Error collecting data frames", ok)
					return
				}
				data = append(data, df.data...)
				if df.isFIN() {
					endflag = 1
				}
				break

			case <-deadline:
				//unsuccessfully waited for FIN
				// no activity in a while. Assume that body is completely recieved. Bail
				//panic("Waited long enough but no data frames recieved")
				debug.Println("Waited long enough but no data frames recieved")
				endflag = 2
				break
			}
		}
		if endflag == 1 {
			//http request if data frames collected sucessfully
			// call the handler
			contLen, _ := strconv.Atoi(headers.Get(HEADER_CONTENT_LENGTH))
			req := &http.Request{
				Method:        headers.Get(HEADER_METHOD),
				Proto:         headers.Get(HEADER_VERSION),
				Header:        headers,
				RemoteAddr:    s.session.conn.RemoteAddr().String(),
				ContentLength: int64(contLen),
				Body:          &readCloser{bytes.NewReader(data)},
			}
			req.URL, _ = url.ParseRequestURI(headers.Get(HEADER_PATH))

			// Clear the headers in the session now that the request has them
			s.headers = make(http.Header)

			go s.requestHandler(req)
		}

	}

	return nil
}
func (s *Stream) requestHandler(req *http.Request) {
	// call the handler - this writes the SYN_REPLY and all data frames
	s.session.server.Handler.ServeHTTP(s, req)

	debug.Printf("Sending final DATA with FIN the handler for #%d", s.id)

	// send an empty data frame with FIN set to end the deal
	frame := dataFrame{stream: s.id, flags: FLAG_FIN}
	s.session.out <- frame

	// close shop for this stream's end
	if !s.closed {
		s.stop_server <- true
	}
}

func (s *Stream) serve() {

	debug.Printf("Stream #%d main loop", s.id)
	err := s.stream_loop()
	if err != nil {
		debug.Println("ERROR in stream loop:", err)
	}
	s.closed = true

	deadline := time.After(1500 * time.Millisecond)
	select {
	case s.session.end_stream <- s:
		// done, all good!
	case <-deadline:
		// somehow it was locked
		debug.Printf("Stream #%d: timed out and cannot be removed from the session", s.id)
	}

	if s.upstream_buffer != nil {
		// this is not a server stream
		close(s.upstream_buffer)
	}
	close(s.flow_add)
	close(s.flow_req)
	debug.Printf("Stream #%d main loop done", s.id)
}

// stream server loop
func (s *Stream) stream_loop() (err error) {

	for {
		deadline := time.After(10 * time.Second)
		select {
		case cf, ok := <-s.control:
			if !ok {
				return
			}
			switch cf.kind {
			case FRAME_SYN_STREAM:
				err = s.initiate_stream(cf)
				debug.Println("Goroutines:", runtime.NumGoroutine())
			case FRAME_SYN_REPLY:
				err = s.handleSynReply(cf)
			case FRAME_RST_STREAM:
				err = s.handleRstStream(cf)
				return
			case FRAME_WINDOW_UPDATE:
				s.handleWindowUpdate(cf)
			default:
				panic("TODO: unhandled type of frame received in stream.serve()")
			}
		case df, ok := <-s.data:
			if !ok {
				return
			}
			err = s.handleDataFrame(df)
		case <-deadline:
			// no activity in a while. bail
			return
		case _, _ = <-s.stop_server:
			return
		}
		if err != nil {
			// finish the loop with any error, data or control
			return
		}
	}
}

// Header makes streams compatible with the net/http handlers interface
func (s *Stream) Header() http.Header { return s.headers }

// Write makes streams compatible with the net/http handlers interface
func (s *Stream) Write(p []byte) (n int, err error) {
	if s.closed {
		err = errors.New(fmt.Sprintf("Stream #%d: write on closed stream!", s.id))
		return
	}
	if !s.wroteHeader {
		s.WriteHeader(http.StatusOK)
	}
	lp := int32(len(p))
	if lp == 0 {
		return
	}
	flow := int32(0)
	for lp > flow {
		// if there is data to send and it's larger than the flow control window
		// we need to stall until we have enough to go
		window, ok := <-s.flow_req
		debug.Printf("Stream #%d: got %d bytes of flow", s.id, window)
		if !ok || s.closed {
			debug.Printf("Stream #%d: flow closed!", s.id)
			return 0, errors.New(fmt.Sprintf("Stream #%d closed while writing"))
		}
		flow += window
	}
	// this is just in case we end up trying to write while on network turbulence
	defer no_panics()
	for len(p) > 0 {
		frame := dataFrame{stream: s.id}
		if len(p) < MAX_DATA_PAYLOAD {
			frame.data = make([]byte, len(p))
		} else {
			frame.data = make([]byte, MAX_DATA_PAYLOAD)
		}
		copy(frame.data, p)
		p = p[len(frame.data):]
		s.session.out <- frame
		n += len(frame.data)
	}

	// put the rest back in the flow control window
	s.flow_add <- flow - int32(n)
	debug.Printf("Stream #%d: FCW updated -%d: %d -> %d", s.id, int32(n), flow, flow-int32(n))

	return
}

// WriteHeader makes streams compatible with the net/http handlers interface
func (s *Stream) WriteHeader(code int) {
	if s.wroteHeader {
		log.Println("ERROR: Multiple calls to ResponseWriter.WriteHeader.")
		return
	}

	// send basic SPDY fields
	s.headers.Set(HEADER_STATUS, strconv.Itoa(code)+" "+http.StatusText(code))
	s.headers.Set(HEADER_VERSION, "HTTP/1.1")

	if s.headers.Get("Content-Type") == "" {
		s.headers.Set("Content-Type", "text/html; charset=utf-8")
	}
	if s.headers.Get("Date") == "" {
		s.headers.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}
	// Write the frame
	sr := frameSynReply{session: s.session, stream: s.id, headers: s.headers}
	debug.Println("Sending SYN_REPLY", sr)
	s.session.out <- sr
	s.wroteHeader = true
}

// takes a SYN_REPLY control frame
func (s *Stream) handleSynReply(frame controlFrame) (err error) {

	debug.Println("Stream server got SYN_REPLY")

	s.headers, err = s.session.headerReader.decode(frame.data[4:])
	if err != nil {
		return
	}
	h := s.response_writer.Header()
	for name, values := range s.headers {
		if name[0] == ':' { // skip SPDY headers
			continue
		}
		for _, value := range values {
			debug.Printf("Header: %s -> %s\n", name, value)
			h.Set(name, value)
		}
	}
	status := s.headers.Get(HEADER_STATUS)
	code, err := strconv.Atoi(status[0:3])
	if err != nil {
		log.Println("ERROR: handleSynReply: got an unparseable status:", status)
	}
	debug.Printf("Header status code: %d\n", code)

	// *could* this conceivably block or time out?
	// we would need to write it through a similar the upstream data sender
	// but remember to NOT update the delta window with this data
	s.response_writer.WriteHeader(code)

	if frame.isFIN() {
		debug.Println("Stream FIN found in SYN_REPLY frame")
		s.eos <- true
	}

	return nil
}

// send stream cancellation
func (s *Stream) sendRstStream() {
	data := new(bytes.Buffer)
	binary.Write(data, binary.BigEndian, s.id)
	// FIXME this needs some cleaning
	var code uint32 = 5 // CANCEL
	binary.Write(data, binary.BigEndian, code)

	rst_stream := controlFrame{kind: FRAME_RST_STREAM, data: data.Bytes()}
	s.session.out <- rst_stream
}

// takes a DATA frame and adds it to the running body of the stream
func (s *Stream) handleDataFrame(frame dataFrame) (err error) {

	debug.Println("Stream server got DATA")

	if len(s.upstream_buffer) >= NORTHBOUND_SLOTS {
		msg := fmt.Sprintf("upstream buffering hit the limit of %d buffers", NORTHBOUND_SLOTS)
		log.Println(msg)
		err = errors.New(msg)
		return
	}

	debug.Printf("Stream #%d adding +%d to upstream data queue. FIN? %v", s.id, len(frame.data), frame.isFIN())
	s.upstream_buffer <- upstream_data{frame.data, frame.isFIN()}
	debug.Printf("Stream #%d data queue size: %d", s.id, len(s.upstream_buffer))

	return
}

func (s *Stream) northboundBufferSender() {
	defer no_panics()
	for f := range s.upstream_buffer {
		var err error
		data := f.data
		size := len(data)
		for l := size; l > 0; l = len(data) {
			debug.Printf("Stream #%d trying to write %d upstream bytes", s.id, l)
			written, err := s.response_writer.Write(data)
			if err == nil && written == l {
				// sunny day scenario!
				break
			}
			if err != nil {
				if !isBrokenPipe(err) {
					log.Printf("ERROR found writing northbound stream #%d:, %#v", s.id, err)
				}
				s.sendRstStream()
				s.eos <- true
				return
			}
			if written != l {
				debug.Printf("Stream #%d: northboundBufferSender: only %d of %d were written", s.id, written, l)
				time.Sleep(2 * time.Second)
			}
			data = data[written:]
		}
		// all good with this write
		if size > 0 {
			debug.Printf("Stream #%d: %d bytes successfully written upstream", s.id, size)
			wupdate := windowUpdateFor(s.id, size)
			s.session.out <- wupdate
			s.session.out <- windowUpdateFor(0, size) // update session window
		}
		if err == nil && f.final {
			debug.Printf("Stream #%d: last upstream data done!", s.id)
			s.eos <- true
			break
		}
	}
	debug.Printf("Stream #%d: northboundBufferSender done!", s.id)
}

// Close does nothing and is here only to allow the data of a request to become
// the body of a response
func (r *readCloser) Close() error { return nil }

func (s *Stream) handleRstStream(frame controlFrame) (err error) {

	debug.Println("Stream server got RST_STREAM")

	id := frame.streamID()

	data := bytes.NewBuffer(frame.data[4:])
	var status uint32
	err = binary.Read(data, binary.BigEndian, &status)
	if err != nil {
		return err
	}
	debug.Printf("Stream #%d cancelled with status code %d", id, status)

	return nil
}

// handle WINDOW_UPDATE from the other side
func (s *Stream) handleWindowUpdate(frame controlFrame) {

	debug.Println("Stream server got WINDOW_UPDATE")

	if s.closed {
		return
	}

	data := bytes.NewBuffer(frame.data[4:8])
	var size uint32
	binary.Read(data, binary.BigEndian, &size)
	size &= 0x7fffffff

	// add the window size update from the flow control window
	s.flow_add <- int32(size)
	debug.Printf("Stream #%d window size +%d", s.id, int32(size))
}

// flowManager is a coroutine to manage the flow control window in an atomic manner
// so that there are no race conditions and it's easier to expand later w/ SETTINGS
func (s *Stream) flowManager(initial int32, in <-chan int32, out chan<- int32) {
	debug.Printf("Stream #%d flow manager started", s.id)
	// no panics; it could be that we get clipped trying to send when out is closed
	defer no_panics()
	sfcw := initial
	for {
		if sfcw > 0 {
			debug.Printf("Stream #%d window size %d", s.id, sfcw)
			select {
			case v, ok := <-in:
				if s.closed || !ok {
					return
				}
				sfcw += v
			case out <- sfcw:
				sfcw = 0
			}
		} else {
			debug.Printf("Stream #%d window size %d", s.id, sfcw)
			v, ok := <-in
			if s.closed || !ok {
				return
			}
			sfcw += v
		}
		if s.closed {
			return
		}
	}
	debug.Printf("Stream #%d flow manager done", s.id)
}
