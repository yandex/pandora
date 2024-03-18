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

func NewCallbackWriter(w io.Writer, onWrite func()) WriterFunc {
	return func(p []byte) (n int, err error) {
		onWrite()
		return w.Write(p)
	}
}
