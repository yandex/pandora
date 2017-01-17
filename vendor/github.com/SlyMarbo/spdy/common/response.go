// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Response is used in handling responses; storing
// the data as it's received, and producing an
// http.Response once complete.
//
// Response may be given a Receiver to enable live
// handling of the response data. This is provided
// by setting spdy.Transport.Receiver. Note that
// providing a Receiver disables the default data
// storage so the returned http.Response.Body will
// be empty.
type Response struct {
	StatusCode int

	headerM sync.Mutex
	Header  http.Header

	dataM sync.Mutex
	data  *hybridBuffer

	Request  *http.Request
	Receiver Receiver
}

func NewResponse(request *http.Request, receiver Receiver) *Response {
	resp := new(Response)
	resp.Request = request
	resp.Receiver = receiver
	if receiver == nil {
		resp.data = newHybridBuffer()
	}
	return resp
}

func (r *Response) ReceiveData(req *http.Request, data []byte, finished bool) {
	if r.Receiver != nil {
		r.Receiver.ReceiveData(req, data, finished)
	} else {
		r.dataM.Lock()
		r.data.Write(data)
		r.dataM.Unlock()
	}
}

func (r *Response) ReceiveHeader(req *http.Request, header http.Header) {
	r.headerM.Lock()
	if r.Header == nil {
		r.Header = make(http.Header)
	}
	UpdateHeader(r.Header, header)
	if status := r.Header.Get(":status"); status != "" {
		status = strings.TrimSpace(status)
		if i := strings.Index(status, " "); i >= 0 {
			status = status[:i]
		}
		s, err := strconv.Atoi(status)
		if err == nil {
			r.StatusCode = s
		}
	}
	if r.Receiver != nil {
		r.Receiver.ReceiveHeader(req, header)
	}
	r.headerM.Unlock()
}

func (r *Response) ReceiveRequest(req *http.Request) bool {
	if r.Receiver != nil {
		return r.Receiver.ReceiveRequest(req)
	}
	return false
}

func (r *Response) Response() *http.Response {
	out := new(http.Response)

	r.headerM.Lock()
	out.Status = fmt.Sprintf("%d %s", r.StatusCode, http.StatusText(r.StatusCode))
	out.StatusCode = r.StatusCode
	out.Header = r.Header
	r.headerM.Unlock()

	out.Proto = "HTTP/1.1"
	out.ProtoMajor = 1
	out.ProtoMinor = 1

	r.dataM.Lock()
	if r.data == nil {
		out.Body = &ReadCloser{new(bytes.Buffer)}
	} else if unrequestedGzip(r) {
		// User-agents MUST support gzip compression.
		// Regardless of the Accept-Encoding sent by the user-agent, the server may
		// always send content encoded with gzip or deflate encoding.
		r.data.Prep()
		out.Header.Del("Content-Encoding")
		out.Header.Del("Content-Length")
		out.ContentLength = -1
		out.Body = &gzipReader{body: r.data}
	} else {
		r.data.Prep()
		out.Body = r.data
		out.ContentLength = r.data.written
	}
	r.dataM.Unlock()

	out.TransferEncoding = nil
	out.Close = true
	out.Trailer = make(http.Header)
	out.Request = r.Request
	return out
}

// 10 MB
var _MAX_MEM_STORAGE = 10 * 1024 * 1024

// hybridBuffer is used in Response to make sure that
// large volumes of data can be stored safely.
type hybridBuffer struct {
	io.Reader

	buf     *bytes.Buffer
	file    *os.File
	written int64
}

func newHybridBuffer() *hybridBuffer {
	hb := new(hybridBuffer)
	hb.buf = new(bytes.Buffer)
	hb.Reader = hb.buf
	return hb
}

func (h *hybridBuffer) Close() error {
	h.buf.Reset()
	if h.file != nil {
		err := h.file.Close()
		if err != nil {
			return err
		}
		return os.Remove(h.file.Name())
	}
	return nil
}

func (h *hybridBuffer) Prep() error {
	if h.file != nil {
		name := h.file.Name()
		err := h.file.Close()
		if err != nil {
			return err
		}
		h.file, err = os.Open(name)
		if err != nil {
			return err
		}
		h.Reader = io.MultiReader(h.buf, h.file)
	}
	return nil
}

func (h *hybridBuffer) Write(b []byte) (int, error) {
	buffered := h.buf.Len()
	var err error

	// Straight to memory
	if len(b)+buffered < _MAX_MEM_STORAGE {
		n, err := h.buf.Write(b)
		h.written += int64(n)
		return n, err
	}

	// Partially to disk
	if buffered < _MAX_MEM_STORAGE {
		mem := _MAX_MEM_STORAGE - buffered
		n, err := h.buf.Write(b[:mem])
		h.written += int64(n)
		if err != nil {
			return n, err
		}
		if h.file == nil {
			h.file, err = ioutil.TempFile("", "spdy_content")
			if err != nil {
				return n, err
			}
			h.Reader = io.MultiReader(h.buf, h.file)
		}
		m, err := h.file.Write(b[mem:])
		h.written += int64(m)
		return m + n, err
	}

	// Fully to disk
	if h.file == nil {
		h.file, err = ioutil.TempFile("", "spdy_content")
		if err != nil {
			return 0, err
		}
		h.Reader = io.MultiReader(h.buf, h.file)
	}
	n, err := h.file.Write(b)
	h.written += int64(n)
	return n, err
}

// unrequestedGzip returns true iff the request did
// not ask for the returned content encoding and that
// encoding is gzip or deflate, which is allowed in
// the SPDY spec.
func unrequestedGzip(r *Response) bool {
	got := r.Header.Get("Content-Encoding")
	switch got {
	case "gzip", "deflate":
	default:
		return false
	}

	requested := r.Request.Header.Get("Accept-Encoding")
	return !strings.Contains(requested, got)
}

// gzipReader wraps a response body so it can lazily
// call gzip.NewReader on the first call to Read
type gzipReader struct {
	body io.ReadCloser // underlying Response.Body
	zr   io.Reader     // lazily-initialized gzip reader
}

func (gz *gzipReader) Read(p []byte) (n int, err error) {
	if gz.zr == nil {
		gz.zr, err = gzip.NewReader(gz.body)
		if err != nil {
			return 0, err
		}
	}
	return gz.zr.Read(p)
}

func (gz *gzipReader) Close() error {
	return gz.body.Close()
}
