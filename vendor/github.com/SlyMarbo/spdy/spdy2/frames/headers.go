// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frames

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/SlyMarbo/spdy/common"
)

type HEADERS struct {
	Flags     common.Flags
	StreamID  common.StreamID
	Header    http.Header
	rawHeader []byte
}

func (frame *HEADERS) Compress(com common.Compressor) error {
	if frame.rawHeader != nil {
		return nil
	}

	data, err := com.Compress(frame.Header)
	if err != nil {
		return err
	}

	frame.rawHeader = data
	return nil
}

func (frame *HEADERS) Decompress(decom common.Decompressor) error {
	if frame.Header != nil {
		return nil
	}

	header, err := decom.Decompress(frame.rawHeader)
	if err != nil {
		return err
	}

	frame.Header = header
	frame.rawHeader = nil
	return nil
}

func (frame *HEADERS) Name() string {
	return "HEADERS"
}

func (frame *HEADERS) ReadFrom(reader io.Reader) (int64, error) {
	c := common.ReadCounter{R: reader}
	data, err := common.ReadExactly(&c, 16)
	if err != nil {
		return c.N, err
	}

	err = controlFrameCommonProcessing(data[:5], _HEADERS, common.FLAG_FIN)
	if err != nil {
		return c.N, err
	}

	// Get and check length.
	length := int(common.BytesToUint24(data[5:8]))
	if length < 6 {
		return c.N, common.IncorrectDataLength(length, 6)
	} else if length > common.MAX_FRAME_SIZE-8 {
		return c.N, common.FrameTooLarge
	}

	// Read in data.
	header, err := common.ReadExactly(&c, length-8)
	if err != nil {
		return c.N, err
	}

	frame.Flags = common.Flags(data[4])
	frame.StreamID = common.StreamID(common.BytesToUint32(data[8:12]))
	frame.rawHeader = header

	if !frame.StreamID.Valid() {
		return c.N, common.StreamIdTooLarge
	}
	if frame.StreamID.Zero() {
		return c.N, common.StreamIdIsZero
	}

	return c.N, nil
}

func (frame *HEADERS) String() string {
	buf := new(bytes.Buffer)

	Flags := ""
	if frame.Flags.FIN() {
		Flags += " common.FLAG_FIN"
	}
	if Flags == "" {
		Flags = "[NONE]"
	} else {
		Flags = Flags[1:]
	}

	buf.WriteString("HEADERS {\n\t")
	buf.WriteString(fmt.Sprintf("Version:              2\n\t"))
	buf.WriteString(fmt.Sprintf("Flags:                %s\n\t", Flags))
	buf.WriteString(fmt.Sprintf("Stream ID:            %d\n\t", frame.StreamID))
	buf.WriteString(fmt.Sprintf("Header:               %v\n}\n", frame.Header))

	return buf.String()
}

func (frame *HEADERS) WriteTo(writer io.Writer) (int64, error) {
	c := common.WriteCounter{W: writer}
	if frame.rawHeader == nil {
		return c.N, errors.New("Error: Headers not written.")
	}
	if !frame.StreamID.Valid() {
		return c.N, common.StreamIdTooLarge
	}
	if frame.StreamID.Zero() {
		return c.N, common.StreamIdIsZero
	}

	header := frame.rawHeader
	length := 4 + len(header)
	out := make([]byte, 16)

	out[0] = 128                  // Control bit and Version
	out[1] = 2                    // Version
	out[2] = 0                    // Type
	out[3] = 8                    // Type
	out[4] = byte(frame.Flags)    // Flags
	out[5] = byte(length >> 16)   // Length
	out[6] = byte(length >> 8)    // Length
	out[7] = byte(length)         // Length
	out[8] = frame.StreamID.B1()  // Stream ID
	out[9] = frame.StreamID.B2()  // Stream ID
	out[10] = frame.StreamID.B3() // Stream ID
	out[11] = frame.StreamID.B4() // Stream ID

	err := common.WriteExactly(&c, out)
	if err != nil {
		return c.N, err
	}

	err = common.WriteExactly(&c, header)
	if err != nil {
		return c.N, err
	}

	return c.N, nil
}
