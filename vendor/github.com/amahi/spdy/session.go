// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// Session related functions

package spdy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync/atomic"
	"time"
)

// NewServerSession creates a new Session with the given network connection.
// This Session should be used as a server, and the given http.Server will be
// used to serve requests arriving.  The user should call Serve() once it's
// ready to start serving. New streams will be created as per the SPDY
// protocol.
func NewServerSession(conn net.Conn, server *http.Server) *Session {
	s := &Session{
		conn:         conn,
		out:          make(chan frame),
		in:           make(chan frame),
		new_stream:   make(chan *Stream),
		end_stream:   make(chan *Stream),
		server:       server,
		headerWriter: newHeaderWriter(),
		headerReader: newHeaderReader(),
		nextStream:   2,
		nextPing:     2,
		streams:      make(map[streamID]*Stream),
		pinger:       make(chan uint32),
	}

	return s
}

// NewClientSession creates a new Session that should be used as a client.
// the given http.Server will be used to serve requests arriving.  The user
// should call Serve() once it's ready to start serving. New streams will be
// created as per the SPDY protocol.
func NewClientSession(conn net.Conn) *Session {
	s := &Session{
		conn:         conn,
		out:          make(chan frame),
		in:           make(chan frame),
		new_stream:   make(chan *Stream),
		end_stream:   make(chan *Stream),
		server:       nil,
		headerWriter: newHeaderWriter(),
		headerReader: newHeaderReader(),
		nextStream:   1,
		nextPing:     1,
		streams:      make(map[streamID]*Stream),
		pinger:       make(chan uint32),
	}

	return s
}

// Serve starts serving a Session. This implementation of Serve only returns
// when there has been an error condition.
func (s *Session) Serve() (err error) {

	debug.Println("Session server started")

	receiver_done := make(chan bool)
	sender_done := make(chan bool)

	// start frame sender
	go s.frameSender(sender_done, s.out)

	// start frame receiver
	go s.frameReceiver(receiver_done, s.in)

	// start serving loop
	err = s.session_loop(sender_done, receiver_done)
	if err != nil {
		log.Printf("ERROR: %s", netErrorString(err))
	}

	// force removing all existing streams
	for i := range s.streams {
		str := s.streams[i]
		str.finish_stream()
		delete(s.streams, i)
	}

	// close this session
	s.Close()
	debug.Println("Session closed. Session server done.")

	return
}

func (s *Session) session_loop(sender_done, receiver_done <-chan bool) (err error) {
	for {
		select {
		case f := <-s.in:
			// received a frame
			switch frame := f.(type) {
			case controlFrame:
				err = s.processControlFrame(frame)
			case dataFrame:
				err = s.processDataFrame(frame)
			}
			if err != nil {
				return
			}
		case ns, ok := <-s.new_stream:
			// registering a new stream for this session
			if ok {
				s.streams[ns.id] = ns
			} else {
				return
			}
		case os, ok := <-s.end_stream:
			// unregistering a stream from this session
			if ok {
				delete(s.streams, os.id)
			} else {
				return
			}
		case _, _ = <-receiver_done:
			debug.Println("Session receiver is done")
			return
		case _, _ = <-sender_done:
			debug.Println("Session sender is done")
			return
		}
	}
}

// Close closes the Session and the underlaying network connection.
// It should be called when the Session is idle for best results.
func (s *Session) Close() {
	// FIXME - what else do we need to do here?
	if s.closed {
		debug.Println("WARNING: session was already closed - why?")
		return
	}

	s.closed = true

	// in case any of the closes below clashes
	defer no_panics()

	close(s.out)
	close(s.in)
	close(s.pinger)

	debug.Println("Closing the network connection")
	s.conn.Close()
}

// return the next stream id
func (s *Session) nextStreamID() streamID {
	return (streamID)(atomic.AddUint32((*uint32)(&s.nextStream), 2) - 2)
}

// frameSender takes a channel and gets each of the frames coming from
// it and sends them down the session connection, until the channel
// is closed or there are errors in sending over the network
func (s *Session) frameSender(done chan<- bool, in <-chan frame) {
	for f := range in {
		s.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_, err := f.Write(s.conn)
		if err != nil {
			log.Println("ERROR in frameSender.Write:", err)
			break
		}
	}
	done <- true
	debug.Printf("Session sender ended")
}

// frameReceiver takes a channel and receives frames, sending them to
// the network connection until there is an error
func (s *Session) frameReceiver(done chan<- bool, incoming chan<- frame) {
	defer no_panics()

	for {
		frame, err := readFrame(s.conn)
		if err == io.EOF {
			// normal reasons, like disconnection, etc.
			break
		}
		if err != nil {
			// some other communication error
			log.Printf("WARN: communication error: %s", netErrorString(err))
			break
		}
		// ship the frame upstream -- this must be ensured to not block
		debug.Printf("Session got: %s", frame)
		incoming <- frame
	}
	done <- true
	debug.Printf("Session receiver ended")
}
func (s *Session) processControlFrame(frame controlFrame) (err error) {

	switch frame.kind {
	case FRAME_SYN_STREAM:
		err = s.processSynStream(frame)
		select {
		case ns, ok := <-s.new_stream:
			// registering a new stream for this session
			if ok {
				s.streams[ns.id] = ns
			} else {
				return
			}
		}
		controlflag := 0
		// for non-FIN frames wait for data frames
		if !frame.isFIN() {
			for controlflag == 0 {
				deadline := time.After(3 * time.Second)
				select {
				case f := <-s.in:
					// received a frame
					switch fr := f.(type) {
					case dataFrame:
						//process data frames
						err = s.processDataFrame(fr)
						if fr.isFIN() {
							controlflag = 1
						}
						break
					case controlFrame:
						err = s.processControlFrame(fr)

					}
					if err != nil {
						return
					}
					break
				case <-deadline:
					//unsuccessfully waited for FIN
					debug.Println("Waited long enough but no data frames recieved")
					controlflag = 2
					break
				}
			}
		}
		return
	case FRAME_SYN_REPLY:
		return s.processSynReply(frame)
	case FRAME_SETTINGS:
		s.processSettings(frame)
		return nil
	case FRAME_RST_STREAM:
		// just to avoid locking issues, send it in a goroutine
		go s.processRstStream(frame)
	case FRAME_PING:
		return s.processPing(frame)
	case FRAME_WINDOW_UPDATE:
		s.processWindowUpdate(frame)
	case FRAME_GOAWAY:
		s.processGoaway(frame)
	case FRAME_HEADERS:
		panic("FIXME HEADERS")
	}

	return
}
func (s *Session) SendGoaway(f frameFlags, dat []byte) {
	s.out <- controlFrame{kind: FRAME_GOAWAY, flags: f, data: dat}
}

func (s *Session) processGoaway(frame controlFrame) {
	if len(frame.data) != 8 {
		log.Println("ERROR: could not process goaway: Frame should be 8 bits long")
		return
	}
	status_code := bytes.NewBuffer(frame.data[4:8])
	var status int32
	err := binary.Read(status_code, binary.BigEndian, &status)
	if err != nil {
		log.Println("ERROR: Cannot read status code from a goaway frame:", err)
		return
	}

	lst_id := frame.streamID()
	debug.Printf("GOAWAY Frame recieved, Last-good-stream-ID: %d, Status Code: %d", lst_id, status)

	//to check if some stream with ID < Last-good-stream-ID is open
	closeSessionFlag := 0

	//Start going away
	s.goaway_recvd = true

	//Close streams with ID > Last-good-stream-ID
	for id, st := range s.streams {
		if id > lst_id {
			if !st.closed {
				st.finish_stream()
				delete(s.streams, id)
			}
		} else {
			if !st.closed {
				closeSessionFlag = 1
			}
		}
	}

	// Close Session if no remaining streams
	if closeSessionFlag == 0 {
		// maybe close the session? Even if session is not closed from this end, the sender will close it from the other end
	}
}

func (s *Session) processDataFrame(frame dataFrame) (err error) {
	stream, found := s.streams[frame.stream]
	if !found {
		// no error because this could happen if a stream is closed with outstanding data
		debug.Printf("WARN: stream %d not found", frame.stream)
		return
	}
	// send it to the stream for processing. this BETTER NOT BLOCK!
	deadline := time.After(300 * time.Millisecond)
	select {
	case stream.data <- frame:
		// send this data frame to the corresponding stream
	case <-deadline:
		// maybe it closed just before we tried to send it
		debug.Printf("Stream #%d: session timed out while sending northbound data", stream.id)
	}

	return
}

func (s *Session) processSynStream(frame controlFrame) (err error) {
	_, err = s.newServerStream(frame)
	if err != nil {
		log.Printf(fmt.Sprintf("cannot create syn stream frame: %s", err))
		return
	}

	return
}

func (s *Session) processSynReply(frame controlFrame) (err error) {

	debug.Println("Processing SYN_REPLY received")
	id := frame.streamID()
	if id == 0 {
		err = errors.New("Invalid stream ID 0 received")
		return
	}

	stream, ok := s.streams[id]
	if !ok {
		err = errors.New(fmt.Sprintf("Stream with ID %d not found", id))
		log.Printf("ERROR: %s", err)
		return
	}

	// send this control frame to the corresponding stream
	stream.control <- frame
	return
}

// Read details for SETTINGS frame
//FIXME : shows error unexpected EOF when communicating with firefox
func (s *Session) processSettings(frame controlFrame) (err error) {
	s.settings = new(settings)
	data := bytes.NewBuffer(frame.data)
	err = binary.Read(data, binary.BigEndian, &s.settings.count)
	if err != nil {
		return
	}
	s.settings.svp = make([]settingsValuePairs, s.settings.count)
	for i := uint32(0); i < s.settings.count; i++ {
		err = binary.Read(data, binary.BigEndian, &s.settings.svp[i].flags)
		if err != nil {
			return
		}
		err = binary.Read(data, binary.BigEndian, &s.settings.svp[i].id)
		if err != nil {
			return
		}
		err = binary.Read(data, binary.BigEndian, &s.settings.svp[i].value)
		if err != nil {
			return
		}
	}

	return
}

func (s *Session) processRstStream(frame controlFrame) {

	debug.Println("Processing RST_STREAM received")
	id := frame.streamID()
	if id == 0 {
		log.Printf("Session: invalid stream ID 0 received")
		return
	}

	stream, ok := s.streams[id]
	if !ok || (ok && stream.closed) {
		debug.Printf("Window update for unknown stream #%d ignored", id)
		debug.Println("known streams are", s.streams)
		return
	}

	// send this control frame to the corresponding stream
	stream.control <- frame
}

// Read details for PING frame
func (s *Session) processPing(frame controlFrame) (err error) {
	s.settings = new(settings)
	var id uint32
	data := bytes.NewBuffer(frame.data[0:4])
	binary.Read(data, binary.BigEndian, &id)
	debug.Printf("PING #%d", id)

	// check that it's initiated by this end or the other
	if (s.nextPing & 0x00000001) == (uint32(id) & 0x00000001) {
		// the ping received matches our partity, do not reply!
		select {
		case s.pinger <- id:
			// Pingback received successfully
		default:
			// noone was listening
			debug.Println("Pingback discarded (received too late)")
		}
		return
	}

	// send it right back!
	s.out <- frame

	return
}

func no_panics() {
	if v := recover(); v != nil {
		debug.Println("Got a panic:", v)
	}
}

// Ping issues a SPDY PING frame and returns true if it the other side returned
// the PING frame within the duration, else it returns false. NOTE only one
// outstanting ping works in the current implementation.
func (s *Session) Ping(d time.Duration) (pinged bool) {

	// increase the next ping id
	id := atomic.AddUint32((*uint32)(&s.nextPing), 2) - 2

	data := new(bytes.Buffer)
	binary.Write(data, binary.BigEndian, id)

	ping := controlFrame{kind: FRAME_PING, flags: 0, data: data.Bytes()}

	defer no_panics()

	s.out <- ping

	pinged = false

	select {
	case pid, ok := <-s.pinger:
		if ok { // make sure we get the same id we sent back
			if pid == id {
				pinged = true
			}
		}
	case <-time.After(d):
		debug.Printf("Pingback timed out")
		// timeout
	}

	return pinged
}

func (s *Session) processWindowUpdate(frame controlFrame) {

	id := frame.streamID()
	if id == 0 {
		// FIXME - rather than panic, just issue a warning, since some
		// browsers will trigger the panic naturally
		log.Println("WARNING: no support for session flow control yet")
	}

	stream, ok := s.streams[id]
	if !ok {
		debug.Printf("Window update for unknown stream #%d ignored", id)
		return
	}
	if stream.closed {
		debug.Printf("Window update for closed stream #%d ignored", id)
		debug.Println("known streams are", s.streams)
		return
	}

	// just to avoid locking issues, send it in a goroutine, and put a deadline
	go func() {
		deadline := time.After(1200 * time.Millisecond)
		select {
		case stream.control <- frame:
			// send this control frame to the corresponding stream
		case <-deadline:
			// maybe it closed just before we tried to send it
			debug.Printf("Stream #%d: session timed out while sending %s north", stream.id, frame)
		}
	}()
}
