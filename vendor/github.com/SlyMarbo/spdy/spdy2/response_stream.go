// Copyright 2013 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy2

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy2/frames"
)

// ResponseStream is a structure that implements the
// Stream interface. This is used for responding to
// client requests.
type ResponseStream struct {
	sync.Mutex

	shutdownOnce   sync.Once
	conn           *Conn
	streamID       common.StreamID
	requestBody    *bytes.Buffer
	state          *common.StreamState
	output         chan<- common.Frame
	request        *http.Request
	handler        http.Handler
	header         http.Header
	priority       common.Priority
	unidirectional bool
	responseCode   int
	ready          chan struct{}
	stop           chan bool
	wroteHeader    bool
}

func NewResponseStream(conn *Conn, frame *frames.SYN_STREAM, output chan<- common.Frame, handler http.Handler, request *http.Request) *ResponseStream {
	out := new(ResponseStream)
	out.conn = conn
	out.streamID = frame.StreamID
	out.output = output
	out.handler = handler
	if out.handler == nil {
		out.handler = http.DefaultServeMux
	}
	out.request = request
	out.priority = frame.Priority
	out.stop = conn.stop
	out.unidirectional = frame.Flags.UNIDIRECTIONAL()
	out.requestBody = new(bytes.Buffer)
	out.state = new(common.StreamState)
	out.header = make(http.Header)
	out.responseCode = 0
	out.ready = make(chan struct{})
	out.wroteHeader = false
	if frame.Flags.FIN() {
		close(out.ready)
		out.state.CloseThere()
	}
	out.request.Body = &common.ReadCloser{out.requestBody}
	return out
}

/***********************
 * http.ResponseWriter *
 ***********************/

func (s *ResponseStream) Header() http.Header {
	return s.header
}

// Write is the main method with which data is sent.
func (s *ResponseStream) Write(inputData []byte) (int, error) {
	if s.unidirectional {
		return 0, errors.New("Error: Stream is unidirectional.")
	}

	if s.closed() || s.state.ClosedHere() {
		return 0, errors.New("Error: Stream already closed.")
	}

	// Copy the data locally to avoid any pointer issues.
	data := make([]byte, len(inputData))
	copy(data, inputData)

	// Default to 200 response.
	if !s.wroteHeader {
		s.WriteHeader(http.StatusOK)
	}

	// Send any new headers.
	s.writeHeader()

	// Chunk the response if necessary.
	written := 0
	for len(data) > common.MAX_DATA_SIZE {
		dataFrame := new(frames.DATA)
		dataFrame.StreamID = s.streamID
		dataFrame.Data = data[:common.MAX_DATA_SIZE]
		s.output <- dataFrame
		
		data = data[common.MAX_DATA_SIZE:]
		written += common.MAX_DATA_SIZE
	}

	n := len(data)
	if n == 0 {
		return written, nil
	}

	dataFrame := new(frames.DATA)
	dataFrame.StreamID = s.streamID
	dataFrame.Data = data
	s.output <- dataFrame

	return written + n, nil
}

// WriteHeader is used to set the HTTP status code.
func (s *ResponseStream) WriteHeader(code int) {
	if s.unidirectional {
		log.Println("Error: Stream is unidirectional.")
		return
	}

	if s.wroteHeader {
		log.Println("Error: Multiple calls to ResponseWriter.WriteHeader.")
		return
	}

	s.wroteHeader = true
	s.responseCode = code
	s.header.Set("status", strconv.Itoa(code))
	s.header.Set("version", "HTTP/1.1")

	// Create the response SYN_REPLY.
	synReply := new(frames.SYN_REPLY)
	synReply.StreamID = s.streamID
	synReply.Header = common.CloneHeader(s.header)

	// Clear the headers that have been sent.
	for name := range synReply.Header {
		s.header.Del(name)
	}

	// These responses have no body, so close the stream now.
	if code == 204 || code == 304 || code/100 == 1 {
		synReply.Flags = common.FLAG_FIN
		s.state.CloseHere()
	}

	s.output <- synReply
}

/*****************
 * io.Closer *
 *****************/

func (s *ResponseStream) Close() error {
	defer common.Recover()
	s.Lock()
	s.shutdownOnce.Do(s.shutdown)
	s.Unlock()
	return nil
}

func (s *ResponseStream) shutdown() {
	s.writeHeader()
	if s.state != nil {
		s.state.Close()
	}
	if s.requestBody != nil {
		s.requestBody.Reset()
		s.requestBody = nil
	}
	s.conn.requestStreamLimit.Close()
	s.request = nil
	s.handler = nil
	s.stop = nil

	s.conn.streamsLock.Lock()
	delete(s.conn.streams, s.streamID)
	s.conn.streamsLock.Unlock()
}

/**********
 * Stream *
 **********/

func (s *ResponseStream) Conn() common.Conn {
	return s.conn
}

func (s *ResponseStream) ReceiveFrame(frame common.Frame) error {
	s.Lock()
	defer s.Unlock()

	if frame == nil {
		return errors.New("Error: Nil frame received.")
	}

	// Process the frame depending on its type.
	switch frame := frame.(type) {
	case *frames.DATA:
		s.requestBody.Write(frame.Data)
		if frame.Flags.FIN() {
			select {
			case <-s.ready:
			default:
				close(s.ready)
			}
			s.state.CloseThere()
		}

	case *frames.SYN_REPLY:
		common.UpdateHeader(s.header, frame.Header)
		if frame.Flags.FIN() {
			select {
			case <-s.ready:
			default:
				close(s.ready)
			}
			s.state.CloseThere()
		}

	case *frames.HEADERS:
		common.UpdateHeader(s.header, frame.Header)

	case *frames.WINDOW_UPDATE:
		// Ignore.

	default:
		return errors.New(fmt.Sprintf("Received unknown frame of type %T.", frame))
	}

	return nil
}

func (s *ResponseStream) CloseNotify() <-chan bool {
	return s.stop
}

// run is the main control path of
// the stream. It is prepared, the
// registered handler is called,
// and then the stream is cleaned
// up and closed.
func (s *ResponseStream) Run() error {
	// Catch any panics.
	defer func() {
		if v := recover(); v != nil {
			if s != nil && s.state != nil && !s.state.Closed() {
				log.Printf("Encountered stream error: %v (%[1]T)\n", v)
			}
		}
	}()

	// Make sure Request is prepared.
	if s.requestBody == nil || s.request.Body == nil {
		s.requestBody = new(bytes.Buffer)
		s.request.Body = &common.ReadCloser{s.requestBody}
	}

	// Wait until the full request has been received.
	<-s.ready

	/***************
	 *** HANDLER ***
	 ***************/
	s.handler.ServeHTTP(s, s.request)

	// Close the stream with a SYN_REPLY if
	// none has been sent, or an empty DATA
	// frame, if a SYN_REPLY has been sent
	// already.
	// If the stream is already closed at
	// this end, then nothing happens.
	if !s.unidirectional {
		if s.state.OpenHere() && !s.wroteHeader {
			h := s.header
			if h == nil {
				h = make(http.Header)
			}

			h.Set("status", "200")
			h.Set("version", "HTTP/1.1")

			// Create the response SYN_REPLY.
			synReply := new(frames.SYN_REPLY)
			synReply.Flags = common.FLAG_FIN
			synReply.StreamID = s.streamID
			synReply.Header = h

			s.output <- synReply
		} else if s.state.OpenHere() {
			// Create the DATA.
			data := new(frames.DATA)
			data.StreamID = s.streamID
			data.Flags = common.FLAG_FIN
			data.Data = []byte{}

			s.output <- data
		}
	}

	// Clean up state.
	s.state.CloseHere()

	if s.state.Closed() {
		return s.Close()
	}

	return nil
}

func (s *ResponseStream) State() *common.StreamState {
	return s.state
}

func (s *ResponseStream) StreamID() common.StreamID {
	return s.streamID
}

func (s *ResponseStream) closed() bool {
	if s.conn == nil || s.state == nil || s.handler == nil {
		return true
	}
	select {
	case _ = <-s.stop:
		return true
	default:
		return false
	}
}

// writeHeader is used to flush HTTP headers.
func (s *ResponseStream) writeHeader() {
	if len(s.header) == 0 || s.unidirectional {
		return
	}

	// Create the HEADERS frame.
	header := new(frames.HEADERS)
	header.StreamID = s.streamID
	header.Header = common.CloneHeader(s.header)

	// Clear the headers that have been sent.
	for name := range header.Header {
		s.header.Del(name)
	}

	s.output <- header
}

/******************
 * PriorityStream *
 ******************/

func (s *ResponseStream) Priority() common.Priority {
	return s.priority
}
