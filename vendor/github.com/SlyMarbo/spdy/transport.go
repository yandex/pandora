// Copyright 2013 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/SlyMarbo/spdy/common"
)

// A Transport is an HTTP/SPDY http.RoundTripper.
type Transport struct {
	m sync.Mutex

	// Proxy specifies a function to return a proxy for a given
	// Request. If the function returns a non-nil error, the
	// request is aborted with the provided error.
	// If Proxy is nil or returns a nil *URL, no proxy is used.
	Proxy func(*http.Request) (*url.URL, error)

	// Dial specifies the dial function for creating TCP
	// connections.
	// If Dial is nil, net.Dial is used.
	Dial func(network, addr string) (net.Conn, error) // TODO: use

	// TLSClientConfig specifies the TLS configuration to use with
	// tls.Client. If nil, the default configuration is used.
	TLSClientConfig *tls.Config

	// DisableKeepAlives, if true, prevents re-use of TCP connections
	// between different HTTP requests.
	DisableKeepAlives bool

	// DisableCompression, if true, prevents the Transport from
	// requesting compression with an "Accept-Encoding: gzip"
	// request header when the Request contains no existing
	// Accept-Encoding value. If the Transport requests gzip on
	// its own and gets a gzipped response, it's transparently
	// decoded in the Response.Body. However, if the user
	// explicitly requested gzip it is not automatically
	// uncompressed.
	DisableCompression bool

	// MaxIdleConnsPerHost, if non-zero, controls the maximum idle
	// (keep-alive) to keep per-host.  If zero,
	// DefaultMaxIdleConnsPerHost is used.
	MaxIdleConnsPerHost int

	// ResponseHeaderTimeout, if non-zero, specifies the amount of
	// time to wait for a server's response headers after fully
	// writing the request (including its body, if any). This
	// time does not include the time to read the response body.
	ResponseHeaderTimeout time.Duration

	spdyConns map[string]common.Conn   // SPDY connections mapped to host:port.
	tcpConns  map[string]chan net.Conn // Non-SPDY connections mapped to host:port.
	connLimit map[string]chan struct{} // Used to enforce the TCP conn limit.

	// Priority is used to determine the request priority of SPDY
	// requests. If nil, spdy.DefaultPriority is used.
	Priority func(*url.URL) common.Priority

	// Receiver is used to receive the server's response. If left
	// nil, the default Receiver will parse and create a normal
	// Response.
	Receiver common.Receiver

	// PushReceiver is used to receive server pushes. If left nil,
	// pushes will be refused. The provided Request will be that
	// sent with the server push. See Receiver for more detail on
	// its methods.
	PushReceiver common.Receiver
}

// NewTransport gives a simple initialised Transport.
func NewTransport(insecureSkipVerify bool) *Transport {
	return &Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: insecureSkipVerify,
			NextProtos:         npn(),
		},
	}
}

// dial makes the connection to an endpoint.
func (t *Transport) dial(u *url.URL) (conn net.Conn, err error) {

	if t.TLSClientConfig == nil {
		t.TLSClientConfig = &tls.Config{
			NextProtos: npn(),
		}
	} else if t.TLSClientConfig.NextProtos == nil {
		t.TLSClientConfig.NextProtos = npn()
	}

	// Wait for a connection slot to become available.
	<-t.connLimit[u.Host]

	switch u.Scheme {
	case "http":
		conn, err = net.Dial("tcp", u.Host)
	case "https":
		conn, err = tls.Dial("tcp", u.Host, t.TLSClientConfig)
	default:
		err = errors.New(fmt.Sprintf("Error: URL has invalid scheme %q.", u.Scheme))
	}

	if err != nil {
		// The connection never happened, which frees up a slot.
		t.connLimit[u.Host] <- struct{}{}
	}

	return conn, err
}

// doHTTP is used to process an HTTP(S) request, using the TCP connection pool.
func (t *Transport) doHTTP(conn net.Conn, req *http.Request) (*http.Response, error) {
	debug.Printf("Requesting %q over HTTP.\n", req.URL.String())

	// Create the HTTP ClientConn, which handles the
	// HTTP details.
	httpConn := httputil.NewClientConn(conn, nil)
	res, err := httpConn.Do(req)
	if err != nil {
		return nil, err
	}

	if !res.Close {
		t.tcpConns[req.URL.Host] <- conn
	} else {
		// This connection is closing, so another can be used.
		t.connLimit[req.URL.Host] <- struct{}{}
		err = httpConn.Close()
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

// RoundTrip handles the actual request; ensuring a connection is
// made, determining which protocol to use, and performing the
// request.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	u := req.URL

	// Make sure the URL host contains the port.
	if !strings.Contains(u.Host, ":") {
		switch u.Scheme {
		case "http":
			u.Host += ":80"

		case "https":
			u.Host += ":443"
		}
	}

	conn, tcpConn, err := t.process(req)
	if err != nil {
		return nil, err
	}
	if tcpConn != nil {
		return t.doHTTP(tcpConn, req)
	}

	// The connection has now been established.

	debug.Printf("Requesting %q over SPDY.\n", u.String())

	// Determine the request priority.
	var priority common.Priority
	if t.Priority != nil {
		priority = t.Priority(req.URL)
	} else {
		priority = common.DefaultPriority(req.URL)
	}

	res, err := conn.RequestResponse(req, t.Receiver, priority)
	if conn.Closed() {
		t.connLimit[u.Host] <- struct{}{}
	}
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (t *Transport) process(req *http.Request) (common.Conn, net.Conn, error) {
	t.m.Lock()
	defer t.m.Unlock()

	u := req.URL

	// Initialise structures if necessary.
	if t.spdyConns == nil {
		t.spdyConns = make(map[string]common.Conn)
	}
	if t.tcpConns == nil {
		t.tcpConns = make(map[string]chan net.Conn)
	}
	if t.connLimit == nil {
		t.connLimit = make(map[string]chan struct{})
	}
	if t.MaxIdleConnsPerHost == 0 {
		t.MaxIdleConnsPerHost = http.DefaultMaxIdleConnsPerHost
	}
	if _, ok := t.connLimit[u.Host]; !ok {
		limitChan := make(chan struct{}, t.MaxIdleConnsPerHost)
		t.connLimit[u.Host] = limitChan
		for i := 0; i < t.MaxIdleConnsPerHost; i++ {
			limitChan <- struct{}{}
		}
	}

	// Check the non-SPDY connection pool.
	if connChan, ok := t.tcpConns[u.Host]; ok {
		select {
		case tcpConn := <-connChan:
			// Use a connection from the pool.
			return nil, tcpConn, nil
		default:
		}
	} else {
		t.tcpConns[u.Host] = make(chan net.Conn, t.MaxIdleConnsPerHost)
	}

	// Check the SPDY connection pool.
	conn, ok := t.spdyConns[u.Host]
	if !ok || u.Scheme == "http" || (conn != nil && conn.Closed()) {
		tcpConn, err := t.dial(req.URL)
		if err != nil {
			return nil, nil, err
		}

		if tlsConn, ok := tcpConn.(*tls.Conn); !ok {
			// Handle HTTP requests.
			return nil, tcpConn, nil
		} else {
			// Handle HTTPS/SPDY requests.
			state := tlsConn.ConnectionState()

			// Complete handshake if necessary.
			if !state.HandshakeComplete {
				err = tlsConn.Handshake()
				if err != nil {
					return nil, nil, err
				}
			}

			// Verify hostname, unless requested not to.
			if !t.TLSClientConfig.InsecureSkipVerify {
				err = tlsConn.VerifyHostname(req.URL.Host)
				if err != nil {
					// Also try verifying the hostname with/without a port number.
					i := strings.Index(req.URL.Host, ":")
					err = tlsConn.VerifyHostname(req.URL.Host[:i])
					if err != nil {
						return nil, nil, err
					}
				}
			}

			// If a protocol could not be negotiated, assume HTTPS.
			if !state.NegotiatedProtocolIsMutual {
				return nil, tcpConn, nil
			}

			// Scan the list of supported NPN strings.
			supported := false
			for _, proto := range npn() {
				if state.NegotiatedProtocol == proto {
					supported = true
					break
				}
			}

			// Ensure the negotiated protocol is supported.
			if !supported && state.NegotiatedProtocol != "" {
				msg := fmt.Sprintf("Error: Unsupported negotiated protocol %q.", state.NegotiatedProtocol)
				return nil, nil, errors.New(msg)
			}

			// Handle the protocol.
			switch state.NegotiatedProtocol {
			case "http/1.1", "":
				return nil, tcpConn, nil

			case "spdy/3.1":
				newConn, err := NewClientConn(tlsConn, t.PushReceiver, 3, 1)
				if err != nil {
					return nil, nil, err
				}
				go newConn.Run()
				t.spdyConns[u.Host] = newConn
				conn = newConn

			case "spdy/3":
				newConn, err := NewClientConn(tlsConn, t.PushReceiver, 3, 0)
				if err != nil {
					return nil, nil, err
				}
				go newConn.Run()
				t.spdyConns[u.Host] = newConn
				conn = newConn

			case "spdy/2":
				newConn, err := NewClientConn(tlsConn, t.PushReceiver, 2, 0)
				if err != nil {
					return nil, nil, err
				}
				go newConn.Run()
				t.spdyConns[u.Host] = newConn
				conn = newConn
			}
		}
	}

	return conn, nil, nil
}
