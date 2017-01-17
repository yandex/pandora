// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frames

import (
	"bytes"
	"fmt"
	"io"

	"github.com/SlyMarbo/spdy/common"
)

type PING struct {
	PingID uint32
}

func (frame *PING) Compress(comp common.Compressor) error {
	return nil
}

func (frame *PING) Decompress(decomp common.Decompressor) error {
	return nil
}

func (frame *PING) Name() string {
	return "PING"
}

func (frame *PING) ReadFrom(reader io.Reader) (int64, error) {
	c := common.ReadCounter{R: reader}
	data, err := common.ReadExactly(&c, 12)
	if err != nil {
		return c.N, err
	}

	err = controlFrameCommonProcessing(data[:5], _PING, 0)
	if err != nil {
		return c.N, err
	}

	// Get and check length.
	length := int(common.BytesToUint24(data[5:8]))
	if length != 4 {
		return c.N, common.IncorrectDataLength(length, 4)
	}

	frame.PingID = common.BytesToUint32(data[8:12])

	return c.N, nil
}

func (frame *PING) String() string {
	buf := new(bytes.Buffer)

	buf.WriteString("PING {\n\t")
	buf.WriteString(fmt.Sprintf("Version:              2\n\t"))
	buf.WriteString(fmt.Sprintf("Ping ID:              %d\n}\n", frame.PingID))

	return buf.String()
}

func (frame *PING) WriteTo(writer io.Writer) (int64, error) {
	c := common.WriteCounter{W: writer}
	out := make([]byte, 12)

	out[0] = 128                      // Control bit and Version
	out[1] = 2                        // Version
	out[2] = 0                        // Type
	out[3] = 6                        // Type
	out[4] = 0                        // Flags
	out[5] = 0                        // Length
	out[6] = 0                        // Length
	out[7] = 4                        // Length
	out[8] = byte(frame.PingID >> 24) // Ping ID
	out[9] = byte(frame.PingID >> 16) // Ping ID
	out[10] = byte(frame.PingID >> 8) // Ping ID
	out[11] = byte(frame.PingID)      // Ping ID

	err := common.WriteExactly(&c, out)
	if err != nil {
		return c.N, err
	}

	return c.N, nil
}
