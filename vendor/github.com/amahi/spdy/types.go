// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// Data structures and types for the Amahi SPDY library

package spdy

import (
	"bytes"
	"crypto/tls"
	"io"
	"net"
	"net/http"
)

type streamID uint32

// Kinds of control frames
type controlFrameKind uint16

const (
	FRAME_SYN_STREAM    = 0x0001
	FRAME_SYN_REPLY     = 0x0002
	FRAME_RST_STREAM    = 0x0003
	FRAME_SETTINGS      = 0x0004
	FRAME_PING          = 0x0006
	FRAME_GOAWAY        = 0x0007
	FRAME_HEADERS       = 0x0008
	FRAME_WINDOW_UPDATE = 0x0009
)

// Frame flags
type frameFlags uint8

const (
	FLAG_NONE = frameFlags(0x00)
	FLAG_FIN  = frameFlags(0x01)
)

type dataFrame struct {
	stream streamID
	flags  frameFlags
	data   []byte
}

type controlFrame struct {
	kind  controlFrameKind
	flags frameFlags
	data  []byte
}

type frame interface {
	Write(io.Writer) (n int64, err error)
	Flags() frameFlags
	String() string
	Data() []byte
}

type Session struct {
	conn         net.Conn     // the underlying connection
	out          chan frame   // channel to send a frame
	in           chan frame   // channel to receive a frame
	new_stream   chan *Stream // channel to register new streams
	end_stream   chan *Stream // channel to unregister streams
	streams      map[streamID]*Stream
	server       *http.Server // http server for this session
	nextStream   streamID     // the next stream ID
	closed       bool         // is this session closed?
	goaway_recvd bool         // recieved goaway
	headerWriter *headerWriter
	headerReader *headerReader
	settings     *settings
	nextPing     uint32 // the next ping ID
	// channel to send our self-initiated pings
	// Ping() listens for an outstanding ping
	pinger chan uint32
}

type settings struct {
	flags frameFlags
	count uint32
	svp   []settingsValuePairs
}

type settingsValuePairs struct {
	flags uint8
	id    uint32
	value uint32
}

type Stream struct {
	id                streamID
	session           *Session
	priority          uint8
	associated_stream streamID
	headers           http.Header
	response_writer   http.ResponseWriter
	closed            bool
	wroteHeader       bool
	// IMPORTANT, these channels must not block (for long)
	control         chan controlFrame // control frames arrive here
	data            chan dataFrame    // data frames arrive here
	response        chan bool         // http responses sent here
	eos             chan bool         // frame handlers poke this when stream is ending
	stop_server     chan bool         // when stream is closed, to stop the server
	flow_req        chan int32        // control flow requests
	flow_add        chan int32        // control flow additions
	upstream_buffer chan upstream_data
}

type upstream_data struct {
	data  []byte
	final bool
}

type frameSynStream struct {
	session           *Session
	stream            streamID
	priority          uint8
	associated_stream streamID
	header            http.Header
	flags             frameFlags
}

type frameSynReply struct {
	session *Session
	stream  streamID
	headers http.Header
	flags   frameFlags
}

// maximum number of bytes in a frame
const MAX_DATA_PAYLOAD = 1<<24 - 1

const (
	HEADER_STATUS         string = ":status"
	HEADER_VERSION        string = ":version"
	HEADER_PATH           string = ":path"
	HEADER_METHOD         string = ":method"
	HEADER_HOST           string = ":host"
	HEADER_SCHEME         string = ":scheme"
	HEADER_CONTENT_LENGTH string = "Content-Length"
)

type readCloser struct {
	io.Reader
}

// ResponseRecorder is an implementation of http.ResponseWriter that
// is used to get a response.
type ResponseRecorder struct {
	Code        int           // the HTTP response code from WriteHeader
	HeaderMap   http.Header   // the HTTP response headers
	Body        *bytes.Buffer // if non-nil, the bytes.Buffer to append written data to
	wroteHeader bool
}

//spdy client
type Client struct {
	cn net.Conn
	ss *Session
}

//spdy server
type Server struct {
	Handler   http.Handler
	Addr      string
	TLSConfig *tls.Config
	ln        net.Listener
	//channel on which the server passes any new spdy 'Session' structs that get created during its lifetime
	ss_chan chan *Session
}

//spdy conn
type conn struct {
	srv *Server
	ss  *Session
	cn  net.Conn
}
