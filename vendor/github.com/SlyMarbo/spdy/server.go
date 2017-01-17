// Copyright 2013 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy2"
	"github.com/SlyMarbo/spdy/spdy3"
)

// NewServerConn is used to create a SPDY connection, using the given
// net.Conn for the underlying connection, and the given http.Server to
// configure the request serving.
func NewServerConn(conn net.Conn, server *http.Server, version, subversion int) (common.Conn, error) {
	if conn == nil {
		return nil, errors.New("Error: Connection initialised with nil net.conn.")
	}
	if server == nil {
		return nil, errors.New("Error: Connection initialised with nil server.")
	}

	switch version {
	case 3:
		return spdy3.NewConn(conn, server, subversion), nil

	case 2:
		return spdy2.NewConn(conn, server), nil

	default:
		return nil, errors.New("Error: Unsupported SPDY version.")
	}
}

// ListenAndServeTLS listens on the TCP network address addr
// and then calls Serve with handler to handle requests on
// incoming connections.  Handler is typically nil, in which
// case the DefaultServeMux is used. Additionally, files
// containing a certificate and matching private key for the
// server must be provided. If the certificate is signed by
// a certificate authority, the certFile should be the
// concatenation of the server's certificate followed by the
// CA's certificate.
//
// See examples/server/server.go for a simple example server.
func ListenAndServeTLS(addr string, certFile string, keyFile string, handler http.Handler) error {
	npnStrings := npn()
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		TLSConfig: &tls.Config{
			NextProtos: npnStrings,
		},
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	for _, str := range npnStrings {
		switch str {
		case "spdy/2":
			server.TLSNextProto[str] = spdy2.NextProto
		case "spdy/3":
			server.TLSNextProto[str] = spdy3.NextProto
		case "spdy/3.1":
			server.TLSNextProto[str] = spdy3.NextProto1
		}
	}

	return server.ListenAndServeTLS(certFile, keyFile)
}

// ListenAndServeSpdyOnly listens on the TCP network address addr
// and then calls Serve with handler to handle requests on
// incoming connections.  Handler is typically nil, in which
// case the DefaultServeMux is used. Additionally, files
// containing a certificate and matching private key for the
// server must be provided. If the certificate is signed by
// a certificate authority, the certFile should be the
// concatenation of the server's certificate followed by the
// CA's certificate.
//
// IMPORTANT NOTE: Unlike spdy.ListenAndServeTLS, this function
// will ONLY serve SPDY. HTTPS requests are refused.
//
// See examples/spdy_only_server/server.go for a simple example server.
func ListenAndServeSpdyOnly(addr string, certFile string, keyFile string, handler http.Handler) error {
	npnStrings := npn()
	if addr == "" {
		addr = ":https"
	}
	if handler == nil {
		handler = http.DefaultServeMux
	}
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		TLSConfig: &tls.Config{
			NextProtos:   npnStrings,
			Certificates: make([]tls.Certificate, 1),
		},
		TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	for _, str := range npnStrings {
		switch str {
		case "spdy/2":
			server.TLSNextProto[str] = spdy2.NextProto
		case "spdy/3":
			server.TLSNextProto[str] = spdy3.NextProto
		case "spdy/3.1":
			server.TLSNextProto[str] = spdy3.NextProto1
		}
	}

	var err error
	server.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(conn, server.TLSConfig)
	defer tlsListener.Close()

	// Main loop
	var tempDelay time.Duration
	for {
		rw, e := tlsListener.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Printf("Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		go serveSPDY(rw, server)
	}
}

// ListenAndServeSPDYNoNPN creates a server that listens exclusively
// for SPDY and (unlike the rest of the package) will not support
// HTTPS.
func ListenAndServeSPDYNoNPN(addr string, certFile string, keyFile string, handler http.Handler, version, subversion int) error {
	if addr == "" {
		addr = ":https"
	}
	if handler == nil {
		handler = http.DefaultServeMux
	}
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		TLSConfig: &tls.Config{
			Certificates: make([]tls.Certificate, 1),
		},
	}

	var err error
	server.TLSConfig.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	conn, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(conn, server.TLSConfig)
	defer tlsListener.Close()

	// Main loop
	var tempDelay time.Duration
	for {
		rw, e := tlsListener.Accept()
		if e != nil {
			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Printf("Accept error: %v; retrying in %v", e, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return e
		}
		tempDelay = 0
		go serveSPDYNoNPN(rw, server, version, subversion)
	}
}

func serveSPDY(conn net.Conn, srv *http.Server) {
	defer common.Recover()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok { // Only allow TLS connections.
		return
	}

	if d := srv.ReadTimeout; d != 0 {
		conn.SetReadDeadline(time.Now().Add(d))
	}
	if d := srv.WriteTimeout; d != 0 {
		conn.SetWriteDeadline(time.Now().Add(d))
	}
	if err := tlsConn.Handshake(); err != nil {
		return
	}

	tlsState := new(tls.ConnectionState)
	*tlsState = tlsConn.ConnectionState()
	proto := tlsState.NegotiatedProtocol
	if fn := srv.TLSNextProto[proto]; fn != nil {
		fn(srv, tlsConn, nil)
	}
	return
}

func serveSPDYNoNPN(conn net.Conn, srv *http.Server, version, subversion int) {
	defer common.Recover()

	tlsConn, ok := conn.(*tls.Conn)
	if !ok { // Only allow TLS connections.
		return
	}

	if d := srv.ReadTimeout; d != 0 {
		conn.SetReadDeadline(time.Now().Add(d))
	}
	if d := srv.WriteTimeout; d != 0 {
		conn.SetWriteDeadline(time.Now().Add(d))
	}
	if err := tlsConn.Handshake(); err != nil {
		return
	}

	serverConn, err := NewServerConn(tlsConn, srv, version, subversion)
	if err != nil {
		log.Println(err)
		return
	}
	serverConn.Run()
}
