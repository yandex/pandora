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

type WINDOW_UPDATE struct {
	StreamID        common.StreamID
	DeltaWindowSize uint32
	subversion      int
}

func (frame *WINDOW_UPDATE) Compress(comp common.Compressor) error {
	return nil
}

func (frame *WINDOW_UPDATE) Decompress(decomp common.Decompressor) error {
	return nil
}

func (frame *WINDOW_UPDATE) Name() string {
	return "WINDOW_UPDATE"
}

func (frame *WINDOW_UPDATE) ReadFrom(reader io.Reader) (int64, error) {
	c := common.ReadCounter{R: reader}
	data, err := common.ReadExactly(&c, 16)
	if err != nil {
		return c.N, err
	}

	err = controlFrameCommonProcessing(data[:5], _WINDOW_UPDATE, 0)
	if err != nil {
		return c.N, err
	}

	// Get and check length.
	length := int(common.BytesToUint24(data[5:8]))
	if length != 8 {
		return c.N, common.IncorrectDataLength(length, 8)
	}

	frame.StreamID = common.StreamID(common.BytesToUint32(data[8:12]))
	frame.DeltaWindowSize = common.BytesToUint32(data[12:16])

	if !frame.StreamID.Valid() {
		return c.N, common.StreamIdTooLarge
	}
	if frame.StreamID.Zero() && frame.subversion == 0 {
		return c.N, common.StreamIdIsZero
	}
	if frame.DeltaWindowSize > common.MAX_DELTA_WINDOW_SIZE {
		return c.N, errors.New("Error: Delta Window Size too large.")
	}

	return c.N, nil
}

func (frame *WINDOW_UPDATE) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString("WINDOW_UPDATE {\n\t")
	buf.WriteString(fmt.Sprintf("Version:              3\n\t"))
	buf.WriteString(fmt.Sprintf("Stream ID:            %d\n\t", frame.StreamID))
	buf.WriteString(fmt.Sprintf("Delta window size:    %d\n}\n", frame.DeltaWindowSize))

	return buf.String()
}

func (frame *WINDOW_UPDATE) WriteTo(writer io.Writer) (int64, error) {
	c := common.WriteCounter{W: writer}
	out := make([]byte, 16)

	out[0] = 128                                     // Control bit and Version
	out[1] = 3                                       // Version
	out[2] = 0                                       // Type
	out[3] = 9                                       // Type
	out[4] = 0                                       // Flags
	out[5] = 0                                       // Length
	out[6] = 0                                       // Length
	out[7] = 8                                       // Length
	out[8] = frame.StreamID.B1()                     // Stream ID
	out[9] = frame.StreamID.B2()                     // Stream ID
	out[10] = frame.StreamID.B3()                    // Stream ID
	out[11] = frame.StreamID.B4()                    // Stream ID
	out[12] = byte(frame.DeltaWindowSize>>24) & 0x7f // Delta Window Size
	out[13] = byte(frame.DeltaWindowSize >> 16)      // Delta Window Size
	out[14] = byte(frame.DeltaWindowSize >> 8)       // Delta Window Size
	out[15] = byte(frame.DeltaWindowSize)            // Delta Window Size

	err := common.WriteExactly(&c, out)
	if err != nil {
		return c.N, err
	}

	return c.N, nil
}
