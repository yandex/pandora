// Copyright 2013 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"fmt"
	"io"
	"net"
	"net/http"
)

// Connection represents a SPDY connection. The connection should
// be started with a call to Run, which will return once the
// connection has been terminated. The connection can be ended
// early by using Close.
type Conn interface {
	http.CloseNotifier
	Close() error
	Closed() bool
	Conn() net.Conn
	Request(request *http.Request, receiver Receiver, priority Priority) (Stream, error)
	RequestResponse(request *http.Request, receiver Receiver, priority Priority) (*http.Response, error)
	Run() error
}

// Stream contains a single SPDY stream.
type Stream interface {
	http.CloseNotifier
	http.ResponseWriter
	Close() error
	Conn() Conn
	ReceiveFrame(Frame) error
	Run() error
	State() *StreamState
	StreamID() StreamID
}

// PushStream contains a single SPDY push stream.
type PushStream interface {
	Stream

	// Fin is used to close the
	// push stream once writing
	// has finished.
	Finish()
}

// PriorityStream represents a SPDY stream with a priority.
type PriorityStream interface {
	Stream

	// Priority returns the stream's
	// priority.
	Priority() Priority
}

// Frame represents a single SPDY frame.
type Frame interface {
	fmt.Stringer
	io.ReaderFrom
	io.WriterTo
	Compress(Compressor) error
	Decompress(Decompressor) error
	Name() string
}

// Compressor is used to compress the text header of a SPDY frame.
type Compressor interface {
	io.Closer
	Compress(http.Header) ([]byte, error)
}

// Decompressor is used to decompress the text header of a SPDY frame.
type Decompressor interface {
	Decompress([]byte) (http.Header, error)
}

// Pinger represents something able to send and
// receive PING frames.
type Pinger interface {
	Ping() (<-chan bool, error)
}

// Pusher represents something able to send
// server puhes.
type Pusher interface {
	Push(url string, origin Stream) (PushStream, error)
}

// SetFlowController represents a connection
// which can have its flow control mechanism
// customised.
type SetFlowController interface {
	SetFlowControl(FlowControl)
}

// Objects implementing the Receiver interface can be
// registered to receive requests on the Client.
//
// ReceiveData is passed the original request, the data
// to receive and a bool indicating whether this is the
// final batch of data. If the bool is set to true, the
// data may be empty, but should not be nil.
//
// ReceiveHeaders is passed the request and any sent
// text headers. This may be called multiple times.
// Note that these headers may contain the status code
// of the response, under the ":status" header. If the
// Receiver is being used to proxy a request, and the
// headers presented to ReceiveHeader are copied to
// another ResponseWriter, take care to call its
// WriteHeader method after copying all headers, since
// this may flush headers received so far.
//
// ReceiveRequest is used when server pushes are sent.
// The returned bool should inticate whether to accept
// the push. The provided Request will be that sent by
// the server with the push.
type Receiver interface {
	ReceiveData(request *http.Request, data []byte, final bool)
	ReceiveHeader(request *http.Request, header http.Header)
	ReceiveRequest(request *http.Request) bool
}

// Objects conforming to the FlowControl interface can be
// used to provide the flow control mechanism for a
// connection using SPDY version 3 and above.
//
// InitialWindowSize is called whenever a new stream is
// created, and the returned value used as the initial
// flow control window size. Note that a values smaller
// than the default (65535) will likely result in poor
// network utilisation.
//
// ReceiveData is called whenever a stream's window is
// consumed by inbound data. The stream's ID is provided,
// along with the stream's initial window size and the
// current window size after receiving the data that
// caused the call. If the window is to be regrown,
// ReceiveData should return the increase in size. A value
// of 0 does not change the window. Note that in SPDY/3.1
// and later, the streamID may be 0 to represent the
// connection-level flow control window.
type FlowControl interface {
	InitialWindowSize() uint32
	ReceiveData(streamID StreamID, initialWindowSize uint32, newWindowSize int64) (deltaSize uint32)
}
