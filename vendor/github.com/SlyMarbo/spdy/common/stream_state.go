// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"sync"
)

// State variables used internally in StreamState.
const (
	stateOpen uint8 = iota
	stateHalfClosedHere
	stateHalfClosedThere
	stateClosed
)

// StreamState is used to store and query the stream's state. The active methods
// do not directly affect the stream's state, but it will use that information
// to effect the changes.
type StreamState struct {
	l sync.Mutex
	s uint8
}

// Check whether the stream is open.
func (s *StreamState) Open() bool {
	s.l.Lock()
	open := s.s == stateOpen
	s.l.Unlock()
	return open
}

// Check whether the stream is closed.
func (s *StreamState) Closed() bool {
	s.l.Lock()
	closed := s.s == stateClosed
	s.l.Unlock()
	return closed
}

// Check whether the stream is half-closed at the other endpoint.
func (s *StreamState) ClosedThere() bool {
	s.l.Lock()
	closedThere := s.s == stateClosed || s.s == stateHalfClosedThere
	s.l.Unlock()
	return closedThere
}

// Check whether the stream is open at the other endpoint.
func (s *StreamState) OpenThere() bool {
	return !s.ClosedThere()
}

// Check whether the stream is half-closed at the other endpoint.
func (s *StreamState) ClosedHere() bool {
	s.l.Lock()
	closedHere := s.s == stateClosed || s.s == stateHalfClosedHere
	s.l.Unlock()
	return closedHere
}

// Check whether the stream is open locally.
func (s *StreamState) OpenHere() bool {
	return !s.ClosedHere()
}

// Closes the stream.
func (s *StreamState) Close() {
	s.l.Lock()
	s.s = stateClosed
	s.l.Unlock()
}

// Half-close the stream locally.
func (s *StreamState) CloseHere() {
	s.l.Lock()
	if s.s == stateOpen {
		s.s = stateHalfClosedHere
	} else if s.s == stateHalfClosedThere {
		s.s = stateClosed
	}
	s.l.Unlock()
}

// Half-close the stream at the other endpoint.
func (s *StreamState) CloseThere() {
	s.l.Lock()
	if s.s == stateOpen {
		s.s = stateHalfClosedThere
	} else if s.s == stateHalfClosedHere {
		s.s = stateClosed
	}
	s.l.Unlock()
}

// State description.
func (s *StreamState) String() string {
	var str string
	if s.OpenHere() {
		str = "open here, "
	} else {
		str = "closed here, "
	}
	if s.OpenThere() {
		str += "open there"
	} else {
		str += "closed there"
	}
	return str
}
