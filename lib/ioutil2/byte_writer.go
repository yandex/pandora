// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package ioutil2

import (
	"bufio"
	"bytes"
	"io"
)

type StringWriter interface {
	WriteString(s string) (n int, err error)
}

// ByteWriter represents efficient io.Writer, that don't need buffering.
// Implemented by *bufio.Writer and *bytes.Buffer.
type ByteWriter interface {
	io.Writer
	StringWriter
	io.ByteWriter
	io.ReaderFrom
}

var _ ByteWriter = &bufio.Writer{}
var _ ByteWriter = &bytes.Buffer{}
