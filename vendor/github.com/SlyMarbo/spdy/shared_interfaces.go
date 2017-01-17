package spdy

import (
	"io"
	"net"
	"net/http"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy2"
	"github.com/SlyMarbo/spdy/spdy3"
)

// Connection represents a SPDY connection. The connection should
// be started with a call to Run, which will return once the
// connection has been terminated. The connection can be ended
// early by using Close.
type Conn interface {
	http.CloseNotifier
	Close() error
	Conn() net.Conn
	Request(request *http.Request, receiver common.Receiver, priority common.Priority) (common.Stream, error)
	RequestResponse(request *http.Request, receiver common.Receiver, priority common.Priority) (*http.Response, error)
	Run() error
}

var _ = Conn(&spdy2.Conn{})
var _ = Conn(&spdy3.Conn{})

// Stream contains a single SPDY stream.
type Stream interface {
	http.CloseNotifier
	http.ResponseWriter
	Close() error
	Conn() common.Conn
	ReceiveFrame(common.Frame) error
	Run() error
	State() *common.StreamState
	StreamID() common.StreamID
}

var _ = Stream(&spdy2.PushStream{})
var _ = Stream(&spdy2.RequestStream{})
var _ = Stream(&spdy2.ResponseStream{})
var _ = Stream(&spdy3.PushStream{})
var _ = Stream(&spdy3.RequestStream{})
var _ = Stream(&spdy3.ResponseStream{})

// PriorityStream represents a SPDY stream with a priority.
type PriorityStream interface {
	Stream

	// Priority returns the stream's
	// priority.
	Priority() common.Priority
}

var _ = PriorityStream(&spdy2.ResponseStream{})
var _ = PriorityStream(&spdy3.ResponseStream{})

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

var _ = Pinger(&spdy2.Conn{})
var _ = Pinger(&spdy3.Conn{})

// Pusher represents something able to send
// server puhes.
type Pusher interface {
	Push(url string, origin common.Stream) (common.PushStream, error)
}

var _ = Pusher(&spdy2.Conn{})
var _ = Pusher(&spdy3.Conn{})

// SetFlowController represents a connection
// which can have its flow control mechanism
// customised.
type SetFlowController interface {
	SetFlowControl(common.FlowControl)
}

var _ = SetFlowController(&spdy3.Conn{})
