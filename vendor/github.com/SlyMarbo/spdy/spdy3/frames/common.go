// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frames

import (
	"bufio"
	"errors"
	"fmt"

	"github.com/SlyMarbo/spdy/common"
)

// ReadFrame reads and parses a frame from reader.
func ReadFrame(reader *bufio.Reader, subversion int) (frame common.Frame, err error) {
	start, err := reader.Peek(4)
	if err != nil {
		return nil, err
	}

	if start[0] != 128 {
		frame = new(DATA)
		_, err = frame.ReadFrom(reader)
		return frame, err
	}

	switch common.BytesToUint16(start[2:4]) {
	case _SYN_STREAM:
		switch subversion {
		case 0:
			frame = new(SYN_STREAM)
		case 1:
			frame = new(SYN_STREAMV3_1)
		default:
			return nil, fmt.Errorf("Error: Given subversion %d is unrecognised.", subversion)
		}
	case _SYN_REPLY:
		frame = new(SYN_REPLY)
	case _RST_STREAM:
		frame = new(RST_STREAM)
	case _SETTINGS:
		frame = new(SETTINGS)
	case _PING:
		frame = new(PING)
	case _GOAWAY:
		frame = new(GOAWAY)
	case _HEADERS:
		frame = new(HEADERS)
	case _WINDOW_UPDATE:
		frame = &WINDOW_UPDATE{subversion: subversion}
	case _CREDENTIAL:
		frame = new(CREDENTIAL)

	default:
		return nil, errors.New("Error Failed to parse frame type.")
	}

	_, err = frame.ReadFrom(reader)
	return frame, err
}

// controlFrameCommonProcessing performs checks identical between
// all control frames. This includes the control bit, the version
// number, the type byte (which is checked against the byte
// provided), and the flags (which are checked against the bitwise
// OR of valid flags provided).
func controlFrameCommonProcessing(data []byte, frameType uint16, flags byte) error {
	// Check it's a control frame.
	if data[0] != 128 {
		return common.IncorrectFrame(_DATA_FRAME, int(frameType), 3)
	}

	// Check version.
	version := (uint16(data[0]&0x7f) << 8) + uint16(data[1])
	if version != 3 {
		return common.UnsupportedVersion(version)
	}

	// Check its type.
	realType := common.BytesToUint16(data[2:])
	if realType != frameType {
		return common.IncorrectFrame(int(realType), int(frameType), 3)
	}

	// Check the flags.
	if data[4] & ^flags != 0 {
		return common.InvalidField("flags", int(data[4]), int(flags))
	}

	return nil
}

// Frame types in SPDY/3
const (
	_SYN_STREAM    = 1
	_SYN_REPLY     = 2
	_RST_STREAM    = 3
	_SETTINGS      = 4
	_PING          = 6
	_GOAWAY        = 7
	_HEADERS       = 8
	_WINDOW_UPDATE = 9
	_CREDENTIAL    = 10
	_CONTROL_FRAME = -1
	_DATA_FRAME    = -2
)

// frameNames provides the name for a particular SPDY/3
// frame type.
var frameNames = map[int]string{
	_SYN_STREAM:    "SYN_STREAM",
	_SYN_REPLY:     "SYN_REPLY",
	_RST_STREAM:    "RST_STREAM",
	_SETTINGS:      "SETTINGS",
	_PING:          "PING",
	_GOAWAY:        "GOAWAY",
	_HEADERS:       "HEADERS",
	_WINDOW_UPDATE: "WINDOW_UPDATE",
	_CREDENTIAL:    "CREDENTIAL",
	_CONTROL_FRAME: "CONTROL_FRAME",
	_DATA_FRAME:    "DATA_FRAME",
}
