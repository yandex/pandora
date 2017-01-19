// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// client connection related functions

package spdy

import (
	"bytes"
	"errors"
	"net"
	"net/http"
	"time"
)

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

// NewRecorder returns an initialized ResponseRecorder.
func NewRecorder() *ResponseRecorder {
	return &ResponseRecorder{
		HeaderMap: make(http.Header),
		Body:      new(bytes.Buffer),
		Code:      200,
	}
}

// Header returns the response headers.
func (rw *ResponseRecorder) Header() http.Header {
	m := rw.HeaderMap
	if m == nil {
		m = make(http.Header)
		rw.HeaderMap = m
	}
	return m
}

// Write always succeeds and writes to rw.Body.
func (rw *ResponseRecorder) Write(buf []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(200)
	}
	if rw.Body != nil {
		len, err := rw.Body.Write(buf)
		return len, err
	} else {
		rw.Body = new(bytes.Buffer)
		len, err := rw.Body.Write(buf)
		return len, err
	}
	return len(buf), nil
}

// WriteHeader sets rw.Code.
func (rw *ResponseRecorder) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.Code = code
	}
	rw.wroteHeader = true
}

//returns a client that reads and writes on c
func NewClientConn(c net.Conn) (*Client, error) {
	session := NewClientSession(c)
	go session.Serve()
	return &Client{cn: c, ss: session}, nil
}

//returns a client with tcp connection created using net.Dial
func NewClient(addr string) (*Client, error) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return &Client{}, err
	}
	session := NewClientSession(conn)
	go session.Serve()
	return &Client{cn: conn, ss: session}, nil
}

//to get a response from the client
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	rr := NewRecorder()
	err := c.ss.NewStreamProxy(req, rr)
	if err != nil {
		return &http.Response{}, err
	}
	resp := &http.Response{
		StatusCode:    rr.Code,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          &readCloser{rr.Body},
		ContentLength: int64(rr.Body.Len()),
		Header:        rr.Header(),
	}
	return resp, nil
}

func (c *Client) Close() error {
	if c.cn == nil {
		err := errors.New("No connection to close")
		return err
	}
	err := c.cn.Close()
	return err
}
func (c *Client) Ping(d time.Duration) (pinged bool, err error) {
	if c.cn == nil {
		err := errors.New("No connection estabilished to server")
		return false, err
	}
	ping := c.ss.Ping(d)
	return ping, nil
}
