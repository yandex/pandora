// Copyright 2013 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy3

import (
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy3/frames"
)

// PushStream is a structure that implements the
// Stream and PushWriter interfaces. this is used
// for performing server pushes.
type PushStream struct {
	sync.Mutex

	shutdownOnce sync.Once
	conn         *Conn
	streamID     common.StreamID
	flow         *flowControl
	origin       common.Stream
	state        *common.StreamState
	output       chan<- common.Frame
	header       http.Header
	stop         <-chan bool
}

func NewPushStream(conn *Conn, streamID common.StreamID, origin common.Stream, output chan<- common.Frame) *PushStream {
	out := new(PushStream)
	out.conn = conn
	out.streamID = streamID
	out.origin = origin
	out.output = output
	out.stop = conn.stop
	out.state = new(common.StreamState)
	out.header = make(http.Header)
	return out
}

/***********************
 * http.ResponseWriter *
 ***********************/

func (p *PushStream) Header() http.Header {
	return p.header
}

// Write is used for sending data in the push.
func (p *PushStream) Write(inputData []byte) (int, error) {
	if p.closed() || p.state.ClosedHere() {
		return 0, errors.New("Error: Stream already closed.")
	}

	state := p.origin.State()
	if p.origin == nil || state.ClosedHere() {
		return 0, errors.New("Error: Origin stream is closed.")
	}

	p.writeHeader()

	// Copy the data locally to avoid any pointer issues.
	data := make([]byte, len(inputData))
	copy(data, inputData)

	// Chunk the response if necessary.
	// Data is sent to the flow control to
	// ensure that the protocol is followed.
	written := 0
	for len(data) > common.MAX_DATA_SIZE {
		n, err := p.flow.Write(data[:common.MAX_DATA_SIZE])
		if err != nil {
			return written, err
		}
		written += n
		data = data[common.MAX_DATA_SIZE:]
	}

	n, err := p.flow.Write(data)
	written += n

	return written, err
}

// WriteHeader is provided to satisfy the Stream
// interface, but has no effect.
func (p *PushStream) WriteHeader(int) {
	p.writeHeader()
	return
}

/*****************
 * io.Closer *
 *****************/

func (p *PushStream) Close() error {
	defer common.Recover()
	p.Lock()
	p.shutdownOnce.Do(p.shutdown)
	p.Unlock()
	return nil
}

func (p *PushStream) shutdown() {
	p.writeHeader()
	if p.state != nil {
		p.state.Close()
	}
	if p.flow != nil {
		p.flow.Close()
	}
	p.conn.pushStreamLimit.Close()
	p.origin = nil
	p.output = nil
	p.header = nil
	p.stop = nil
}

/**********
 * Stream *
 **********/

func (p *PushStream) Conn() common.Conn {
	return p.conn
}

func (p *PushStream) ReceiveFrame(frame common.Frame) error {
	p.Lock()
	defer p.Unlock()

	if frame == nil {
		return errors.New("Error: Nil frame received.")
	}

	// Process the frame depending on its type.
	switch frame := frame.(type) {
	case *frames.WINDOW_UPDATE:
		err := p.flow.UpdateWindow(frame.DeltaWindowSize)
		if err != nil {
			reply := new(frames.RST_STREAM)
			reply.StreamID = p.streamID
			reply.Status = common.RST_STREAM_FLOW_CONTROL_ERROR
			p.output <- reply
			return err
		}

	default:
		return errors.New(fmt.Sprintf("Received unexpected frame of type %T.", frame))
	}

	return nil
}

func (p *PushStream) CloseNotify() <-chan bool {
	return p.stop
}

func (p *PushStream) Run() error {
	return nil
}

func (p *PushStream) State() *common.StreamState {
	return p.state
}

func (p *PushStream) StreamID() common.StreamID {
	return p.streamID
}

/**************
 * PushStream *
 **************/

func (p *PushStream) Finish() {
	p.writeHeader()
	end := new(frames.DATA)
	end.StreamID = p.streamID
	end.Data = []byte{}
	end.Flags = common.FLAG_FIN
	p.output <- end
	p.Close()
}

/**********
 * Others *
 **********/

func (p *PushStream) closed() bool {
	if p.conn == nil || p.state == nil {
		return true
	}
	select {
	case _ = <-p.stop:
		return true
	default:
		return false
	}
}

// writeHeader is used to send HTTP headers to
// the client.
func (p *PushStream) writeHeader() {
	if len(p.header) == 0 || p.closed() {
		return
	}

	header := new(frames.HEADERS)
	header.StreamID = p.streamID
	header.Header = make(http.Header)

	for name, values := range p.header {
		for _, value := range values {
			header.Header.Add(name, value)
		}
		p.header.Del(name)
	}

	if len(header.Header) == 0 {
		return
	}

	p.output <- header
}
