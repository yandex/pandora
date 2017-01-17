// Copyright 2013 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy2

import (
	"bufio"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy2/frames"
)

// Conn is a spdy.Conn implementing SPDY/2. This is used in both
// servers and clients, and is created with either NewServerConn,
// or NewClientConn.
type Conn struct {
	PushReceiver common.Receiver // Receiver to call for server Pushes.

	// network state
	remoteAddr  string
	server      *http.Server                      // nil if client connection.
	conn        net.Conn                          // underlying network (TLS) connection.
	connLock    sync.Mutex                        // protects the interface value of the above conn.
	buf         *bufio.Reader                     // buffered reader on conn.
	tlsState    *tls.ConnectionState              // underlying TLS connection state.
	streams     map[common.StreamID]common.Stream // map of active streams.
	streamsLock sync.Mutex                        // protects streams.
	output      [8]chan common.Frame              // one output channel per priority level.

	// other state
	compressor       common.Compressor   // outbound compression state.
	decompressor     common.Decompressor // inbound decompression state.
	receivedSettings common.Settings     // settings sent by client.
	goawayReceived   bool                // goaway has been received.
	goawaySent       bool                // goaway has been sent.
	goawayLock       sync.Mutex          // protects goawaySent and goawayReceived.
	numBenignErrors  int                 // number of non-serious errors encountered.
	readTimeout      time.Duration       // optional timeout for network reads.
	writeTimeout     time.Duration       // optional timeout for network writes.
	timeoutLock      sync.Mutex          // protects changes to readTimeout and writeTimeout.

	// SPDY features
	pings                map[uint32]chan<- bool                // response channel for pings.
	pingsLock            sync.Mutex                            // protects pings.
	nextPingID           uint32                                // next outbound ping ID.
	nextPingIDLock       sync.Mutex                            // protects nextPingID.
	pushStreamLimit      *common.StreamLimit                   // Limit on streams started by the server.
	pushRequests         map[common.StreamID]*http.Request     // map of requests sent in server pushes.
	lastPushStreamID     common.StreamID                       // last push stream ID. (even)
	lastPushStreamIDLock sync.Mutex                            // protects lastPushStreamID.
	pushedResources      map[common.Stream]map[string]struct{} // prevents duplicate headers being pushed.

	// requests
	lastRequestStreamID     common.StreamID     // last request stream ID. (odd)
	lastRequestStreamIDLock sync.Mutex          // protects lastRequestStreamID.
	streamCreation          sync.Mutex          // ensures new streams are sent in order.
	oddity                  common.StreamID     // whether locally-sent streams are odd or even.
	initialWindowSize       uint32              // initial transport window.
	initialWindowSizeLock   sync.Mutex          // lock for initialWindowSize
	requestStreamLimit      *common.StreamLimit // Limit on streams started by the client.

	// startup and shutdown
	stop          chan bool     // this channel is closed when the connection closes.
	sending       chan struct{} // this channel is used to ensure pending frames are sent.
	sendingLock   sync.Mutex    // protects changes to sending's value.
	init          func()        // this function is called before the connection begins.
	shutdownOnce  sync.Once     // used to ensure clean shutdown.
	shutdownError error         // error that caused shutdown if non-nil
}

// NewConn produces an initialised spdy3 connection.
func NewConn(conn net.Conn, server *http.Server) *Conn {
	out := new(Conn)

	// Common ground.
	out.remoteAddr = conn.RemoteAddr().String()
	out.server = server
	out.conn = conn
	out.buf = bufio.NewReader(conn)
	if tlsConn, ok := conn.(*tls.Conn); ok {
		out.tlsState = new(tls.ConnectionState)
		*out.tlsState = tlsConn.ConnectionState()
	}
	out.streams = make(map[common.StreamID]common.Stream)
	out.output[0] = make(chan common.Frame)
	out.output[1] = make(chan common.Frame)
	out.output[2] = make(chan common.Frame)
	out.output[3] = make(chan common.Frame)
	out.output[4] = make(chan common.Frame)
	out.output[5] = make(chan common.Frame)
	out.output[6] = make(chan common.Frame)
	out.output[7] = make(chan common.Frame)
	out.pings = make(map[uint32]chan<- bool)
	out.compressor = common.NewCompressor(2)
	out.decompressor = common.NewDecompressor(2)
	out.receivedSettings = make(common.Settings)
	out.lastPushStreamID = 0
	out.lastRequestStreamID = 0
	out.stop = make(chan bool)

	// Server/client specific.
	if server != nil { // servers
		out.nextPingID = 2
		out.oddity = 0
		out.initialWindowSize = common.DEFAULT_INITIAL_WINDOW_SIZE
		out.requestStreamLimit = common.NewStreamLimit(common.DEFAULT_STREAM_LIMIT)
		out.pushStreamLimit = common.NewStreamLimit(common.NO_STREAM_LIMIT)
		out.init = func() {
			// Initialise the connection by sending the connection settings.
			settings := new(frames.SETTINGS)
			settings.Settings = defaultServerSettings(common.DEFAULT_STREAM_LIMIT)
			out.output[0] <- settings
		}
		if d := server.ReadTimeout; d != 0 {
			out.SetReadTimeout(d)
		}
		if d := server.WriteTimeout; d != 0 {
			out.SetWriteTimeout(d)
		}
		out.pushedResources = make(map[common.Stream]map[string]struct{})

	} else { // clients
		out.nextPingID = 1
		out.oddity = 1
		out.initialWindowSize = common.DEFAULT_INITIAL_CLIENT_WINDOW_SIZE
		out.requestStreamLimit = common.NewStreamLimit(common.NO_STREAM_LIMIT)
		out.pushStreamLimit = common.NewStreamLimit(common.DEFAULT_STREAM_LIMIT)
		out.pushRequests = make(map[common.StreamID]*http.Request)
		out.init = func() {
			// Initialise the connection by sending the connection settings.
			settings := new(frames.SETTINGS)
			settings.Settings = defaultClientSettings(common.DEFAULT_STREAM_LIMIT)
			out.output[0] <- settings
		}
	}
	return out
}

// NextProto is intended for use in http.Server.TLSNextProto,
// using SPDY/2 for the connection.
func NextProto(s *http.Server, tlsConn *tls.Conn, handler http.Handler) {
	NewConn(tlsConn, s).Run()
}

func (c *Conn) Run() error {
	defer common.Recover()
	go c.send()        // Start the send loop.
	if c.init != nil { // Must be after sending is enabled.
		c.init() // Prepare any initialisation frames.
	}
	go c.readFrames() // Start the main loop.
	<-c.stop          // Run until the connection ends.
	return nil
}

// newStream is used to create a new serverStream from a SYN_STREAM frame.
func (c *Conn) newStream(frame *frames.SYN_STREAM) *ResponseStream {
	header := frame.Header
	rawUrl := header.Get("scheme") + "://" + header.Get("host") + header.Get("url")

	url, err := url.Parse(rawUrl)
	if c.check(err != nil, "Received SYN_STREAM with invalid request URL (%v)", err) {
		return nil
	}

	vers := header.Get("version")
	major, minor, ok := http.ParseHTTPVersion(vers)
	if c.check(!ok, "Invalid HTTP version: "+vers) {
		return nil
	}

	method := header.Get("method")

	// Build this into a request to present to the Handler.
	request := &http.Request{
		Method:     method,
		URL:        url,
		Proto:      vers,
		ProtoMajor: major,
		ProtoMinor: minor,
		RemoteAddr: c.remoteAddr,
		Header:     header,
		Host:       url.Host,
		RequestURI: url.RequestURI(),
		TLS:        c.tlsState,
	}

	output := c.output[frame.Priority]
	c.streamCreation.Lock()
	out := NewResponseStream(c, frame, output, c.server.Handler, request)
	c.streamCreation.Unlock()

	return out
}
