// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"sync"
)

// StreamLimit is used to add and enforce
// a limit on the number of concurrently
// active streams.
type StreamLimit struct {
	lock    sync.Mutex
	limit   uint32
	current uint32
}

func NewStreamLimit(limit uint32) *StreamLimit {
	out := new(StreamLimit)
	out.limit = limit
	return out
}

// SetLimit is used to modify the stream limit. If the
// limit is set to NO_STREAM_LIMIT, then the limiting
// is disabled.
func (s *StreamLimit) SetLimit(l uint32) {
	s.lock.Lock()
	s.limit = l
	s.lock.Unlock()
}

// Limit returns the current limit.
func (s *StreamLimit) Limit() uint32 {
	return s.limit
}

// Add is called when a new stream is to be opened. Add
// returns a bool indicating whether the stream is safe
// open.
func (s *StreamLimit) Add() bool {
	s.lock.Lock()
	if s.current >= s.limit {
		s.lock.Unlock()
		return false
	}
	s.current++
	s.lock.Unlock()
	return true
}

// Close is called when a stream is closed; thus freeing
// up a slot.
func (s *StreamLimit) Close() {
	s.lock.Lock()
	s.current--
	s.lock.Unlock()
}
