// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// server connection related functions

package spdy

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

func (c *conn) handleConnection(outchan chan *Session) {
	hserve := new(http.Server)
	if c.srv.Handler == nil {
		hserve.Handler = http.DefaultServeMux
	} else {
		hserve.Handler = c.srv.Handler
	}
	hserve.Addr = c.srv.Addr
	c.ss = NewServerSession(c.cn, hserve)
	if outchan != nil {
		outchan <- c.ss
	}
	c.ss.Serve()
}

// ListenAndServe listens on the TCP network address s.Addr and then
// calls Serve to handle requests on incoming connections.
func (s *Server) ListenAndServe() (err error) {
	if s.Addr == "" {
		s.Addr = ":http"
	}
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	return s.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)})
}

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each.  The service goroutines read requests and
// then call srv.Handler to reply to them.
func (s *Server) Serve(ln net.Listener) (err error) {
	s.ln = ln
	defer s.ln.Close()
	var tempDelay time.Duration // how long to sleep on accept failure
	for {
		rw, err := s.ln.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}
				log.Printf("http: Accept error: %v; retrying in %v", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0
		c, err := s.newConn(rw)
		if err != nil {
			continue
		}
		go c.handleConnection(s.ss_chan)
	}
}

//close spdy server and return
// Any blocked Accept operations will be unblocked and return errors.
func (s *Server) Close() (err error) {
	return s.ln.Close()
}

// Create new connection from rw
func (server *Server) newConn(rwc net.Conn) (c *conn, err error) {
	c = &conn{
		srv: server,
		cn:  rwc,
	}
	return c, nil
}

// ListenAndServe listens on the TCP network address addr
// and then calls Serve with handler to handle requests
// on incoming connections.  Handler is typically nil,
// in which case the DefaultServeMux is used. This creates a spdy
// only server without TLS
//
// A trivial example server is:
//
//	package main
//
//	import (
//		"io"
//		"net/http"
//              "github.com/amahi/spdy"
//		"log"
//	)
//
//	// hello world, the web server
//	func HelloServer(w http.ResponseWriter, req *http.Request) {
//		io.WriteString(w, "hello, world!\n")
//	}
//
//	func main() {
//		http.HandleFunc("/hello", HelloServer)
//		err := spdy.ListenAndServe(":12345", nil)
//		if err != nil {
//			log.Fatal("ListenAndServe: ", err)
//		}
//	}
func ListenAndServe(addr string, handler http.Handler) (err error) {
	server := &Server{
		Addr:    addr,
		Handler: handler,
	}
	return server.ListenAndServe()
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Servers created this way have NPN Negotiation and
// accept requests from both spdy and http clients.
// Additionally, files containing a certificate and matching private
// key for the server must be provided. If the certificate is signed by a certificate
// authority, the certFile should be the concatenation of the server's certificate
// followed by the CA's certificate.
//
// A trivial example server is:
//
//	import (
//		"log"
//		"net/http"
//              "github.com/amahi/spdy"
//	)
//
//	func handler(w http.ResponseWriter, req *http.Request) {
//		w.Header().Set("Content-Type", "text/plain")
//		w.Write([]byte("This is an example server.\n"))
//	}
//
//	func main() {
//		http.HandleFunc("/", handler)
//		log.Printf("About to listen on 10443. Go to https://127.0.0.1:10443/")
//		err := spdy.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil)
//		if err != nil {
//			log.Fatal(err)
//		}
//	}
//
// One can use makecert.sh in /certs to generate certfile and keyfile
func ListenAndServeTLS(addr string, certFile string, keyFile string, handler http.Handler) error {
	server := &http.Server{
		Addr:    addr,
		Handler: handler,
		TLSConfig: &tls.Config{
			NextProtos: []string{"spdy/3.1", "spdy/3"},
		},
		TLSNextProto: map[string]func(*http.Server, *tls.Conn, http.Handler){
			"spdy/3.1": nextproto3,
			"spdy/3":   nextproto3,
		},
	}
	if server.Handler == nil {
		server.Handler = http.DefaultServeMux
	}
	return server.ListenAndServeTLS(certFile, keyFile)
}

func nextproto3(s *http.Server, c *tls.Conn, h http.Handler) {
	server_session := NewServerSession(c, s)
	server_session.Serve()
}

func ListenAndServeTLSSpdyOnly(addr string, certFile string, keyFile string, handler http.Handler) error {
	server := &Server{
		Addr:    addr,
		Handler: handler,
	}
	return server.ListenAndServeTLSSpdyOnly(certFile, keyFile)
}

// ListenAndServeTLSSpdyOnly listens on the TCP network address srv.Addr and
// then calls Serve to handle requests on incoming TLS connections.
// This is a spdy-only server with TLS and no NPN.
//
// Filenames containing a certificate and matching private key for
// the server must be provided. If the certificate is signed by a
// certificate authority, the certFile should be the concatenation
// of the server's certificate followed by the CA's certificate.
func (srv *Server) ListenAndServeTLSSpdyOnly(certFile, keyFile string) error {
	addr := srv.Addr
	if addr == "" {
		addr = ":https"
	}
	config := &tls.Config{}
	if srv.TLSConfig != nil {
		*config = *srv.TLSConfig
	}
	if config.NextProtos == nil {
		config.NextProtos = []string{"spdy/3.1", "spdy/3"}
	}
	var err error
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0], err = tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	tlsListener := tls.NewListener(tcpKeepAliveListener{ln.(*net.TCPListener)}, config)
	srv.TLSConfig = config
	return srv.Serve(tlsListener)
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return
	}
	tc.SetKeepAlive(true)
	tc.SetKeepAlivePeriod(3 * time.Minute)
	return tc, nil
}
