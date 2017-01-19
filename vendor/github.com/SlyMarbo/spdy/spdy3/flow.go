// Copyright 2013 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy3

import (
	"errors"
	"sync"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy3/frames"
)

type DefaultFlowControl uint32

func (f DefaultFlowControl) InitialWindowSize() uint32 {
	return uint32(f)
}

func (f DefaultFlowControl) ReceiveData(_ common.StreamID, initialWindowSize uint32, newWindowSize int64) uint32 {
	if newWindowSize < (int64(initialWindowSize) / 2) {
		return uint32(int64(initialWindowSize) - newWindowSize)
	}

	return 0
}

// flowControl is used by Streams to ensure that
// they abide by SPDY's flow control rules. For
// versions of SPDY before 3, this has no effect.
type flowControl struct {
	sync.Mutex
	conn                *Conn
	stream              common.Stream
	streamID            common.StreamID
	output              chan<- common.Frame
	initialWindow       uint32
	transferWindow      int64
	sent                uint32
	buffer              [][]byte
	constrained         bool
	initialWindowThere  uint32
	transferWindowThere int64
	flowControl         common.FlowControl
	waiting             chan bool
}

// AddFlowControl initialises flow control for
// the Stream. If the Stream is running at an
// older SPDY version than SPDY/3, the flow
// control has no effect. Multiple calls to
// AddFlowControl are safe.
func (s *PushStream) AddFlowControl(f common.FlowControl) {
	if s.flow != nil {
		return
	}

	s.flow = new(flowControl)
	s.flow.conn = s.conn
	s.conn.initialWindowSizeLock.Lock()
	initialWindow := s.conn.initialWindowSize
	s.conn.initialWindowSizeLock.Unlock()
	s.flow.streamID = s.streamID
	s.flow.output = s.output
	s.flow.buffer = make([][]byte, 0, 10)
	s.flow.initialWindow = initialWindow
	s.flow.transferWindow = int64(initialWindow)
	s.flow.stream = s
	s.flow.flowControl = f
	s.flow.initialWindowThere = f.InitialWindowSize()
	s.flow.transferWindowThere = int64(s.flow.transferWindowThere)
}

// AddFlowControl initialises flow control for
// the Stream. If the Stream is running at an
// older SPDY version than SPDY/3, the flow
// control has no effect. Multiple calls to
// AddFlowControl are safe.
func (s *RequestStream) AddFlowControl(f common.FlowControl) {
	if s.flow != nil {
		return
	}

	s.flow = new(flowControl)
	s.flow.conn = s.conn
	s.conn.initialWindowSizeLock.Lock()
	initialWindow := s.conn.initialWindowSize
	s.conn.initialWindowSizeLock.Unlock()
	s.flow.streamID = s.streamID
	s.flow.output = s.output
	s.flow.buffer = make([][]byte, 0, 10)
	s.flow.initialWindow = initialWindow
	s.flow.transferWindow = int64(initialWindow)
	s.flow.stream = s
	s.flow.flowControl = f
	s.flow.initialWindowThere = f.InitialWindowSize()
	s.flow.transferWindowThere = int64(s.flow.initialWindowThere)
}

// AddFlowControl initialises flow control for
// the Stream. If the Stream is running at an
// older SPDY version than SPDY/3, the flow
// control has no effect. Multiple calls to
// AddFlowControl are safe.
func (s *ResponseStream) AddFlowControl(f common.FlowControl) {
	if s.flow != nil {
		return
	}

	s.flow = new(flowControl)
	s.flow.conn = s.conn
	s.conn.initialWindowSizeLock.Lock()
	initialWindow := s.conn.initialWindowSize
	s.conn.initialWindowSizeLock.Unlock()
	s.flow.streamID = s.streamID
	s.flow.output = s.output
	s.flow.buffer = make([][]byte, 0, 10)
	s.flow.initialWindow = initialWindow
	s.flow.transferWindow = int64(initialWindow)
	s.flow.stream = s
	s.flow.flowControl = f
	s.flow.initialWindowThere = f.InitialWindowSize()
	s.flow.transferWindowThere = int64(s.flow.initialWindowThere)
}

// CheckInitialWindow is used to handle the race
// condition where the flow control is initialised
// before the server has received any updates to
// the initial tranfer window sent by the client.
//
// The transfer window is updated retroactively,
// if necessary.
func (f *flowControl) CheckInitialWindow() {
	if f.stream == nil || f.stream.Conn() == nil {
		return
	}

	f.conn.initialWindowSizeLock.Lock()
	newWindow := f.conn.initialWindowSize
	f.conn.initialWindowSizeLock.Unlock()

	if f.initialWindow != newWindow {
		if f.initialWindow > newWindow {
			f.transferWindow = int64(newWindow - f.sent)
		} else if f.initialWindow < newWindow {
			f.transferWindow += int64(newWindow - f.initialWindow)
		}
		if f.transferWindow <= 0 {
			f.constrained = true
		}
		f.initialWindow = newWindow
	}
}

// Close nils any references held by the flowControl.
func (f *flowControl) Close() {
	f.buffer = nil
	f.stream = nil
}

// Flush is used to send buffered data to
// the connection, if the transfer window
// will allow. Flush does not guarantee
// that any or all buffered data will be
// sent with a single flush.
func (f *flowControl) Flush() {
	f.CheckInitialWindow()
	if !f.constrained || f.transferWindow == 0 {
		return
	}

	out := make([]byte, 0, f.transferWindow)
	left := f.transferWindow
	for i := 0; i < len(f.buffer); i++ {
		if l := int64(len(f.buffer[i])); l <= left {
			out = append(out, f.buffer[i]...)
			left -= l
			f.buffer = f.buffer[1:]
		} else {
			out = append(out, f.buffer[i][:left]...)
			f.buffer[i] = f.buffer[i][left:]
			left = 0
		}

		if left == 0 {
			break
		}
	}

	f.transferWindow -= int64(len(out))

	if f.transferWindow > 0 {
		f.constrained = false
		debug.Printf("Stream %d is no longer constrained.\n", f.streamID)
	}

	dataFrame := new(frames.DATA)
	dataFrame.StreamID = f.streamID
	dataFrame.Data = out

	f.output <- dataFrame
}

// Paused indicates whether there is data buffered.
// A Stream should not be closed until after the
// last data has been sent and then Paused returns
// false.
func (f *flowControl) Paused() bool {
	f.CheckInitialWindow()
	return f.constrained
}

// Receive is called when data is received from
// the other endpoint. This ensures that they
// conform to the transfer window, regrows the
// window, and sends errors if necessary.
func (f *flowControl) Receive(data []byte) {
	// The transfer window shouldn't already be negative.
	if f.transferWindowThere < 0 {
		rst := new(frames.RST_STREAM)
		rst.StreamID = f.streamID
		rst.Status = common.RST_STREAM_FLOW_CONTROL_ERROR
		f.output <- rst
	}

	// Update the window.
	f.transferWindowThere -= int64(len(data))

	// Regrow the window if it's half-empty.
	delta := f.flowControl.ReceiveData(f.streamID, f.initialWindowThere, f.transferWindowThere)
	if delta != 0 {
		grow := new(frames.WINDOW_UPDATE)
		grow.StreamID = f.streamID
		grow.DeltaWindowSize = delta
		f.output <- grow
		f.transferWindowThere += int64(grow.DeltaWindowSize)
	}
}

// UpdateWindow is called when an UPDATE_WINDOW frame is received,
// and performs the growing of the transfer window.
func (f *flowControl) UpdateWindow(deltaWindowSize uint32) error {
	f.Lock()
	defer f.Unlock()

	if int64(deltaWindowSize)+f.transferWindow > common.MAX_TRANSFER_WINDOW_SIZE {
		return errors.New("Error: WINDOW_UPDATE delta window size overflows transfer window size.")
	}

	// Grow window and flush queue.
	debug.Printf("Flow: Growing window in stream %d by %d bytes.\n", f.streamID, deltaWindowSize)
	f.transferWindow += int64(deltaWindowSize)

	f.Flush()
	select {
	case f.waiting <- true:
	default:
	}

	return nil
}

// Wait blocks until any buffered data has been sent.
// This may involve waiting for a window update from
// the peer.
func (f *flowControl) Wait() error {
	f.Lock()
	f.Flush()
	if !f.Paused() {
		f.Unlock()
		return nil
	}

	if f.waiting != nil {
		f.Unlock()
		return errors.New("waiting for flow control twice")
	}

	f.waiting = make(chan bool)
	f.Unlock()

	for {
		<-f.waiting
		f.Flush()
		if !f.Paused() {
			return nil
		}
	}
}

// Write is used to send data to the connection. This
// takes care of the windowing. Although data may be
// buffered, rather than actually sent, this is not
// visible to the caller.
func (f *flowControl) Write(data []byte) (int, error) {
	l := len(data)
	if l == 0 {
		return 0, nil
	}

	if f.buffer == nil || f.stream == nil {
		return 0, errors.New("Error: Stream closed.")
	}

	// Transfer window processing.
	f.CheckInitialWindow()
	if f.constrained {
		f.Flush()
	}

	f.Lock()
	var window uint32
	if f.transferWindow < 0 {
		window = 0
	} else {
		window = uint32(f.transferWindow)
	}

	constrained := false
	sending := uint32(len(data))
	if sending > window {
		sending = window
		constrained = true
	}

	f.sent += sending
	f.transferWindow -= int64(sending)

	if constrained {
		f.buffer = append(f.buffer, data[window:])
		data = data[:window]
		f.constrained = true
		debug.Printf("Stream %d is now constrained.\n", f.streamID)
	}
	f.Unlock()

	if len(data) == 0 {
		return l, nil
	}

	dataFrame := new(frames.DATA)
	dataFrame.StreamID = f.streamID
	dataFrame.Data = data

	f.output <- dataFrame
	return l, nil
}
