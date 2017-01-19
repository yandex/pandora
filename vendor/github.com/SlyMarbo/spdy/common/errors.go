// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
)

// MaxBenignErrors is the maximum number of minor errors each
// connection will allow without ending the session.
//
// By default, MaxBenignErrors is set to 0, disabling checks
// and allowing minor errors to go unchecked, although they
// will still be reported to the debug logger. If it is
// important that no errors go unchecked, such as when testing
// another implementation, set MaxBenignErrors to 1 or higher.
var MaxBenignErrors = 0

var (
	ErrConnNil        = errors.New("Error: Connection is nil.")
	ErrConnClosed     = errors.New("Error: Connection is closed.")
	ErrGoaway         = errors.New("Error: GOAWAY received.")
	ErrNoFlowControl  = errors.New("Error: This connection does not use flow control.")
	ErrConnectFail    = errors.New("Error: Failed to connect.")
	ErrInvalidVersion = errors.New("Error: Invalid SPDY version.")

	// ErrNotSPDY indicates that a SPDY-specific feature was attempted
	// with a ResponseWriter using a non-SPDY connection.
	ErrNotSPDY = errors.New("Error: Not a SPDY connection.")

	// ErrNotConnected indicates that a SPDY-specific feature was
	// attempted with a Client not connected to the given server.
	ErrNotConnected = errors.New("Error: Not connected to given server.")
)

type incorrectDataLength struct {
	got, expected int
}

func IncorrectDataLength(got, expected int) error {
	return &incorrectDataLength{got, expected}
}

func (i *incorrectDataLength) Error() string {
	return fmt.Sprintf("Error: Incorrect amount of data for frame: got %d bytes, expected %d.", i.got, i.expected)
}

var FrameTooLarge = errors.New("Error: Frame too large.")

type invalidField struct {
	field         string
	got, expected int
}

func InvalidField(field string, got, expected int) error {
	return &invalidField{field, got, expected}
}

func (i *invalidField) Error() string {
	return fmt.Sprintf("Error: Field %q recieved invalid data %d, expecting %d.", i.field, i.got, i.expected)
}

type incorrectFrame struct {
	got, expected, version int
}

func IncorrectFrame(got, expected, version int) error {
	return &incorrectFrame{got, expected, version}
}

func (i *incorrectFrame) Error() string {
	if i.version == 3 {
		return fmt.Sprintf("Error: Frame %s tried to parse data for a %s.", frameNamesV3[i.expected], frameNamesV3[i.got])
	}
	return fmt.Sprintf("Error: Frame %s tried to parse data for a %s.", frameNamesV2[i.expected], frameNamesV2[i.got])
}

var StreamIdTooLarge = errors.New("Error: Stream ID is too large.")

var StreamIdIsZero = errors.New("Error: Stream ID is zero.")

type UnsupportedVersion uint16

func (u UnsupportedVersion) Error() string {
	return fmt.Sprintf("Error: Unsupported SPDY version: %d.\n", u)
}

func Recover() {
	v := recover()
	if v == nil {
		return
	}

	log.Printf("spdy: panic: %v (%[1]T)\n", v)
	for skip := 1; ; skip++ {
		pc, file, line, ok := runtime.Caller(skip)
		if ok {
			f := runtime.FuncForPC(pc)
			if filepath.Ext(file) != ".go" {
				continue
			}

			log.Printf("- %s:%d in %s()\n", file, line, f.Name())
			if f.Name() == "main.main" {
				return
			}
		} else {
			log.Println("- ???:? in ???()")
			return
		}
	}
}
