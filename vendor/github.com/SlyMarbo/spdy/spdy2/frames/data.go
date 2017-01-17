// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frames

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/SlyMarbo/spdy/common"
)

type DATA struct {
	StreamID common.StreamID
	Flags    common.Flags
	Data     []byte
}

func (frame *DATA) Compress(comp common.Compressor) error {
	return nil
}

func (frame *DATA) Decompress(decomp common.Decompressor) error {
	return nil
}

func (frame *DATA) Name() string {
	return "DATA"
}

func (frame *DATA) ReadFrom(reader io.Reader) (int64, error) {
	c := common.ReadCounter{R: reader}
	data, err := common.ReadExactly(&c, 8)
	if err != nil {
		return c.N, err
	}

	// Check it's a data frame.
	if data[0]&0x80 == 1 {
		return c.N, common.IncorrectFrame(_CONTROL_FRAME, _DATA_FRAME, 2)
	}

	// Check flags.
	if data[4] & ^byte(common.FLAG_FIN) != 0 {
		return c.N, common.InvalidField("flags", int(data[4]), common.FLAG_FIN)
	}

	// Get and check length.
	length := int(common.BytesToUint24(data[5:8]))
	if length > common.MAX_FRAME_SIZE-8 {
		return c.N, common.FrameTooLarge
	}

	// Read in data.
	if length != 0 {
		frame.Data, err = common.ReadExactly(&c, length)
		if err != nil {
			return c.N, err
		}
	}

	frame.StreamID = common.StreamID(common.BytesToUint32(data[0:4]))
	frame.Flags = common.Flags(data[4])
	if frame.Data == nil {
		frame.Data = []byte{}
	}

	return c.N, nil
}

func (frame *DATA) String() string {
	buf := new(bytes.Buffer)

	flags := ""
	if frame.Flags.FIN() {
		flags += " common.FLAG_FIN"
	}
	if flags == "" {
		flags = "[NONE]"
	} else {
		flags = flags[1:]
	}

	buf.WriteString("DATA {\n\t")
	buf.WriteString(fmt.Sprintf("Stream ID:            %d\n\t", frame.StreamID))
	buf.WriteString(fmt.Sprintf("Flags:                %s\n\t", flags))
	buf.WriteString(fmt.Sprintf("Length:               %d\n\t", len(frame.Data)))
	if common.VerboseLogging || len(frame.Data) <= 21 {
		buf.WriteString(fmt.Sprintf("Data:                 [% x]\n}\n", frame.Data))
	} else {
		buf.WriteString(fmt.Sprintf("Data:                 [% x ... % x]\n}\n", frame.Data[:9],
			frame.Data[len(frame.Data)-9:]))
	}

	return buf.String()
}

func (frame *DATA) WriteTo(writer io.Writer) (int64, error) {
	c := common.WriteCounter{W: writer}
	length := len(frame.Data)
	if length > common.MAX_DATA_SIZE {
		return c.N, errors.New("Error: Data size too large.")
	}
	if length == 0 && !frame.Flags.FIN() {
		return c.N, errors.New("Error: Data is empty.")
	}

	out := make([]byte, 8)

	out[0] = frame.StreamID.B1() // Control bit and Stream ID
	out[1] = frame.StreamID.B2() // Stream ID
	out[2] = frame.StreamID.B3() // Stream ID
	out[3] = frame.StreamID.B4() // Stream ID
	out[4] = byte(frame.Flags)   // Flags
	out[5] = byte(length >> 16)  // Length
	out[6] = byte(length >> 8)   // Length
	out[7] = byte(length)        // Length

	err := common.WriteExactly(&c, out)
	if err != nil {
		return c.N, err
	}

	err = common.WriteExactly(&c, frame.Data)
	if err != nil {
		return c.N, err
	}

	return c.N, nil
}
