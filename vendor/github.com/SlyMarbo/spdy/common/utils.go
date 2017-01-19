// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"io"
	"net/http"
)

// CloneHeader returns a duplicate of the provided Header.
func CloneHeader(h http.Header) http.Header {
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2
	}
	return h2
}

// UpdateHeader adds and new name/value pairs and replaces
// those already existing in the older header.
func UpdateHeader(older, newer http.Header) {
	for name, values := range newer {
		for i, value := range values {
			if i == 0 {
				older.Set(name, value)
			} else {
				older.Add(name, value)
			}
		}
	}
}

func BytesToUint16(b []byte) uint16 {
	return (uint16(b[0]) << 8) + uint16(b[1])
}

func BytesToUint24(b []byte) uint32 {
	return (uint32(b[0]) << 16) + (uint32(b[1]) << 8) + uint32(b[2])
}

func BytesToUint24Reverse(b []byte) uint32 {
	return (uint32(b[2]) << 16) + (uint32(b[1]) << 8) + uint32(b[0])
}

func BytesToUint32(b []byte) uint32 {
	return (uint32(b[0]) << 24) + (uint32(b[1]) << 16) + (uint32(b[2]) << 8) + uint32(b[3])
}

// ReadExactly is used to ensure that the given number of bytes
// are read if possible, even if multiple calls to Read
// are required.
func ReadExactly(r io.Reader, i int) ([]byte, error) {
	out := make([]byte, i)
	in := out[:]
	for i > 0 {
		if r == nil {
			return nil, ErrConnNil
		}
		if n, err := r.Read(in); err != nil {
			return nil, err
		} else {
			in = in[n:]
			i -= n
		}
	}
	return out, nil
}

// WriteExactly is used to ensure that the given data is written
// if possible, even if multiple calls to Write are
// required.
func WriteExactly(w io.Writer, data []byte) error {
	i := len(data)
	for i > 0 {
		if w == nil {
			return ErrConnNil
		}
		if n, err := w.Write(data); err != nil {
			return err
		} else {
			data = data[n:]
			i -= n
		}
	}
	return nil
}

// ReadCloser is a helper structure to allow
// an io.Reader to satisfy the io.ReadCloser
// interface.
type ReadCloser struct {
	io.Reader
}

func (r *ReadCloser) Close() error {
	return nil
}

// ReadCounter is a helper structure for
// keeping track of the number of bytes
// read from an io.Reader
type ReadCounter struct {
	N int64
	R io.Reader
}

func (r *ReadCounter) Read(b []byte) (n int, err error) {
	n, err = r.R.Read(b)
	r.N += int64(n)
	return
}

// WriteCounter is a helper structure for
// keeping track of the number of bytes
// written from an io.Writer
type WriteCounter struct {
	N int64
	W io.Writer
}

func (w *WriteCounter) Write(b []byte) (n int, err error) {
	n, err = w.W.Write(b)
	w.N += int64(n)
	return
}
