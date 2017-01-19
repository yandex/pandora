// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package frames

import (
	"io"

	"github.com/SlyMarbo/spdy/common"
)

type NOOP struct{}

func (frame *NOOP) Compress(comp common.Compressor) error {
	return nil
}

func (frame *NOOP) Decompress(decomp common.Decompressor) error {
	return nil
}

func (frame *NOOP) Name() string {
	return "NOOP"
}

func (frame *NOOP) ReadFrom(reader io.Reader) (int64, error) {
	c := common.ReadCounter{R: reader}
	data, err := common.ReadExactly(&c, 8)
	if err != nil {
		return c.N, err
	}

	err = controlFrameCommonProcessing(data[:5], _NOOP, 0)
	if err != nil {
		return c.N, err
	}

	// Get and check length.
	length := int(common.BytesToUint24(data[5:8]))
	if length != 0 {
		return c.N, common.IncorrectDataLength(length, 0)
	}

	return c.N, nil
}

func (frame *NOOP) String() string {
	return "NOOP {\n\tVersion:              2\n}\n"
}

func (frame *NOOP) WriteTo(writer io.Writer) (int64, error) {
	return 0, nil
}
