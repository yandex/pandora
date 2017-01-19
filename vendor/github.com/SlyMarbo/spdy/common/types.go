// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"fmt"
	"sort"
)

/************
 * StreamID *
 ************/

// StreamID is the unique identifier for a single SPDY stream.
type StreamID uint32

func (s StreamID) B1() byte {
	return byte(s >> 24)
}

func (s StreamID) B2() byte {
	return byte(s >> 16)
}

func (s StreamID) B3() byte {
	return byte(s >> 8)
}

func (s StreamID) B4() byte {
	return byte(s)
}

// Client indicates whether the ID should belong to a client-sent stream.
func (s StreamID) Client() bool {
	return s != 0 && s&1 != 0
}

// Server indicates whether the ID should belong to a server-sent stream.
func (s StreamID) Server() bool {
	return s != 0 && s&1 == 0
}

// Valid indicates whether the ID is in the range of legal values (including 0).
func (s StreamID) Valid() bool {
	return s <= MAX_STREAM_ID
}

// Zero indicates whether the ID is zero.
func (s StreamID) Zero() bool {
	return s == 0
}

/*********
 * Flags *
 *********/

// Flags represent a frame's Flags.
type Flags byte

// CLEAR_SETTINGS indicates whether the CLEAR_SETTINGS
// flag is set.
func (f Flags) CLEAR_SETTINGS() bool {
	return f&FLAG_SETTINGS_CLEAR_SETTINGS != 0
}

// FIN indicates whether the FIN flag is set.
func (f Flags) FIN() bool {
	return f&FLAG_FIN != 0
}

// PERSIST_VALUE indicates whether the PERSIST_VALUE
// flag is set.
func (f Flags) PERSIST_VALUE() bool {
	return f&FLAG_SETTINGS_PERSIST_VALUE != 0
}

// PERSISTED indicates whether the PERSISTED flag is
// set.
func (f Flags) PERSISTED() bool {
	return f&FLAG_SETTINGS_PERSISTED != 0
}

// UNIDIRECTIONAL indicates whether the UNIDIRECTIONAL
// flag is set.
func (f Flags) UNIDIRECTIONAL() bool {
	return f&FLAG_UNIDIRECTIONAL != 0
}

/************
 * Priority *
 ************/

// Priority represents a stream's priority.
type Priority byte

// Byte returns the priority in binary form, adjusted
// for the given SPDY version.
func (p Priority) Byte(version uint16) byte {
	switch version {
	case 3:
		return byte((p & 7) << 5)
	case 2:
		return byte((p & 3) << 6)
	default:
		return 0
	}
}

// Valid indicates whether the priority is in the valid
// range for the given SPDY version.
func (p Priority) Valid(version uint16) bool {
	switch version {
	case 3:
		return p <= 7
	case 2:
		return p <= 3
	default:
		return false
	}
}

/**************
 * StatusCode *
 **************/

// StatusCode represents a status code sent in
// certain SPDY frames, such as RST_STREAM and
// GOAWAY.
type StatusCode uint32

func (r StatusCode) B1() byte {
	return byte(r >> 24)
}

func (r StatusCode) B2() byte {
	return byte(r >> 16)
}

func (r StatusCode) B3() byte {
	return byte(r >> 8)
}

func (r StatusCode) B4() byte {
	return byte(r)
}

// IsFatal returns a bool indicating
// whether receiving the given status
// code should end the connection.
func (r StatusCode) IsFatal() bool {
	switch r {
	case RST_STREAM_PROTOCOL_ERROR:
		return true
	case RST_STREAM_INTERNAL_ERROR:
		return true
	case RST_STREAM_FRAME_TOO_LARGE:
		return true
	case RST_STREAM_UNSUPPORTED_VERSION:
		return true

	default:
		return false
	}
}

// String gives the StatusCode in text form.
func (r StatusCode) String() string {
	return statusCodeText[r]
}

/************
 * Settings *
 ************/

// Setting represents a single setting as sent
// in a SPDY SETTINGS frame.
type Setting struct {
	Flags Flags
	ID    uint32
	Value uint32
}

// String gives the textual representation of a Setting.
func (s *Setting) String() string {
	id := settingText[s.ID] + ":"
	Flags := ""
	if s.Flags.PERSIST_VALUE() {
		Flags += " FLAG_SETTINGS_PERSIST_VALUE"
	}
	if s.Flags.PERSISTED() {
		Flags += " FLAG_SETTINGS_PERSISTED"
	}
	if Flags == "" {
		Flags = "[NONE]"
	} else {
		Flags = Flags[1:]
	}

	return fmt.Sprintf("%-31s %-10d %s", id, s.Value, Flags)
}

// Settings represents a series of settings, stored in a map
// by setting ID. This ensures that duplicate settings are
// not sent, since the new value will replace the old.
type Settings map[uint32]*Setting

// Settings returns a slice of Setting, sorted into order by
// ID, as in the SPDY specification.
func (s Settings) Settings() []*Setting {
	if len(s) == 0 {
		return []*Setting{}
	}

	ids := make([]int, 0, len(s))
	for id := range s {
		ids = append(ids, int(id))
	}

	sort.Sort(sort.IntSlice(ids))

	out := make([]*Setting, len(s))

	for i, id := range ids {
		out[i] = s[uint32(id)]
	}

	return out
}
