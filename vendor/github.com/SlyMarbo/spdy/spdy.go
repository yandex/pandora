// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"

	"github.com/SlyMarbo/spdy/common"
	"github.com/SlyMarbo/spdy/spdy2"
	"github.com/SlyMarbo/spdy/spdy3"
)

// SetMaxBenignErrors is used to modify the maximum number
// of minor errors each connection will allow without ending
// the session.
//
// By default, the value is set to 0, disabling checks
// and allowing minor errors to go unchecked, although they
// will still be reported to the debug logger. If it is
// important that no errors go unchecked, such as when testing
// another implementation, SetMaxBenignErrors with 1 or higher.
func SetMaxBenignErrors(n int) {
	common.MaxBenignErrors = n
}

// AddSPDY adds SPDY support to srv, and must be called before srv begins serving.
func AddSPDY(srv *http.Server) {
	if srv == nil {
		return
	}

	npnStrings := npn()
	if len(npnStrings) <= 1 {
		return
	}
	if srv.TLSConfig == nil {
		srv.TLSConfig = new(tls.Config)
	}
	if srv.TLSConfig.NextProtos == nil {
		srv.TLSConfig.NextProtos = npnStrings
	} else {
		// Collect compatible alternative protocols.
		others := make([]string, 0, len(srv.TLSConfig.NextProtos))
		for _, other := range srv.TLSConfig.NextProtos {
			if !strings.Contains(other, "spdy/") && !strings.Contains(other, "http/") {
				others = append(others, other)
			}
		}

		// Start with spdy.
		srv.TLSConfig.NextProtos = make([]string, 0, len(others)+len(npnStrings))
		srv.TLSConfig.NextProtos = append(srv.TLSConfig.NextProtos, npnStrings[:len(npnStrings)-1]...)

		// Add the others.
		srv.TLSConfig.NextProtos = append(srv.TLSConfig.NextProtos, others...)
		srv.TLSConfig.NextProtos = append(srv.TLSConfig.NextProtos, "http/1.1")
	}
	if srv.TLSNextProto == nil {
		srv.TLSNextProto = make(map[string]func(*http.Server, *tls.Conn, http.Handler))
	}
	for _, str := range npnStrings {
		switch str {
		case "spdy/2":
			srv.TLSNextProto[str] = spdy2.NextProto
		case "spdy/3":
			srv.TLSNextProto[str] = spdy3.NextProto
		case "spdy/3.1":
			srv.TLSNextProto[str] = spdy3.NextProto1
		}
	}
}

// GetPriority is used to identify the request priority of the
// given stream. This can be used to manually enforce stream
// priority, although this is already performed by the
// library.
// If the underlying connection is using HTTP, and not SPDY,
// GetPriority will return the ErrNotSPDY error.
//
// A simple example of finding a stream's priority is:
//
//      import (
//              "github.com/SlyMarbo/spdy"
//              "log"
//              "net/http"
//      )
//
//      func httpHandler(w http.ResponseWriter, r *http.Request) {
//							priority, err := spdy.GetPriority(w)
//              if err != nil {
//                      // Non-SPDY connection.
//              } else {
//                      log.Println(priority)
//              }
//      }
//
//      func main() {
//              http.HandleFunc("/", httpHandler)
//              log.Printf("About to listen on 10443. Go to https://127.0.0.1:10443/")
//              err := spdy.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil)
//              if err != nil {
//                      log.Fatal(err)
//              }
//      }
func GetPriority(w http.ResponseWriter) (int, error) {
	if stream, ok := w.(PriorityStream); ok {
		return int(stream.Priority()), nil
	}
	return 0, common.ErrNotSPDY
}

// PingClient is used to send PINGs with SPDY servers.
// PingClient takes a ResponseWriter and returns a channel on
// which a spdy.Ping will be sent when the PING response is
// received. If the channel is closed before a spdy.Ping has
// been sent, this indicates that the PING was unsuccessful.
//
// If the underlying connection is using HTTP, and not SPDY,
// PingClient will return the ErrNotSPDY error.
//
// A simple example of sending a ping is:
//
//      import (
//              "github.com/SlyMarbo/spdy"
//              "log"
//              "net/http"
//      )
//
//      func httpHandler(w http.ResponseWriter, req *http.Request) {
//              ping, err := spdy.PingClient(w)
//              if err != nil {
//                      // Non-SPDY connection.
//              } else {
//                      resp, ok <- ping
//                      if ok {
//                              // Ping was successful.
//                      }
//              }
//
//      }
//
//      func main() {
//              http.HandleFunc("/", httpHandler)
//              log.Printf("About to listen on 10443. Go to https://127.0.0.1:10443/")
//              err := spdy.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil)
//              if err != nil {
//                      log.Fatal(err)
//              }
//      }
func PingClient(w http.ResponseWriter) (<-chan bool, error) {
	if stream, ok := w.(Stream); !ok {
		return nil, common.ErrNotSPDY
	} else {
		return stream.Conn().(Pinger).Ping()
	}
}

// PingServer is used to send PINGs with http.Clients using.
// SPDY. PingServer takes a ResponseWriter and returns a
// channel onwhich a spdy.Ping will be sent when the PING
// response is received. If the channel is closed before a
// spdy.Ping has been sent, this indicates that the PING was
// unsuccessful.
//
// If the underlying connection is using HTTP, and not SPDY,
// PingServer will return the ErrNotSPDY error.
//
// If an underlying connection has not been made to the given
// server, PingServer will return the ErrNotConnected error.
//
// A simple example of sending a ping is:
//
//      import (
//              "github.com/SlyMarbo/spdy"
//              "net/http"
//      )
//
//      func main() {
//              resp, err := http.Get("https://example.com/")
//
//              // ...
//
//              ping, err := spdy.PingServer(http.DefaultClient, "https://example.com")
//              if err != nil {
//                      // No SPDY connection.
//              } else {
//                      resp, ok <- ping
//                      if ok {
//                              // Ping was successful.
//                      }
//              }
//      }
func PingServer(c http.Client, server string) (<-chan bool, error) {
	if transport, ok := c.Transport.(*Transport); !ok {
		return nil, common.ErrNotSPDY
	} else {
		u, err := url.Parse(server)
		if err != nil {
			return nil, err
		}
		// Make sure the URL host contains the port.
		if !strings.Contains(u.Host, ":") {
			switch u.Scheme {
			case "http":
				u.Host += ":80"

			case "https":
				u.Host += ":443"
			}
		}
		conn, ok := transport.spdyConns[u.Host]
		if !ok || conn == nil {
			return nil, common.ErrNotConnected
		}
		return conn.(Pinger).Ping()
	}
}

// Push is used to send server pushes with SPDY servers.
// Push takes a ResponseWriter and the url of the resource
// being pushed, and returns a ResponseWriter to which the
// push should be written.
//
// If the underlying connection is using HTTP, and not SPDY,
// Push will return the ErrNotSPDY error.
//
// A simple example of pushing a file is:
//
//      import (
//              "github.com/SlyMarbo/spdy"
//              "log"
//              "net/http"
//      )
//
//      func httpHandler(w http.ResponseWriter, r *http.Request) {
//              path := r.URL.Scheme + "://" + r.URL.Host + "/javascript.js"
//              push, err := spdy.Push(w, path)
//              if err != nil {
//                      // Non-SPDY connection.
//              } else {
//                      http.ServeFile(push, r, "./javascript.js") // Push the given file.
//											push.Finish()                              // Finish the stream once used.
//              }
//
//      }
//
//      func main() {
//              http.HandleFunc("/", httpHandler)
//              log.Printf("About to listen on 10443. Go to https://127.0.0.1:10443/")
//              err := spdy.ListenAndServeTLS(":10443", "cert.pem", "key.pem", nil)
//              if err != nil {
//                      log.Fatal(err)
//              }
//      }
func Push(w http.ResponseWriter, url string) (common.PushStream, error) {
	if stream, ok := w.(Stream); !ok {
		return nil, common.ErrNotSPDY
	} else {
		return stream.Conn().(Pusher).Push(url, stream)
	}
}

// SetFlowControl can be used to set the flow control mechanism on
// the underlying SPDY connection.
func SetFlowControl(w http.ResponseWriter, f common.FlowControl) error {
	if stream, ok := w.(Stream); !ok {
		return common.ErrNotSPDY
	} else if controller, ok := stream.Conn().(SetFlowController); !ok {
		return common.ErrNotSPDY
	} else {
		controller.SetFlowControl(f)
		return nil
	}
}

// SPDYversion returns the SPDY version being used in the underlying
// connection used by the given http.ResponseWriter. This is 0 for
// connections not using SPDY.
func SPDYversion(w http.ResponseWriter) float64 {
	if stream, ok := w.(Stream); ok {
		switch stream := stream.Conn().(type) {
		case *spdy3.Conn:
			switch stream.Subversion {
			case 0:
				return 3
			case 1:
				return 3.1
			default:
				return 0
			}

		case *spdy2.Conn:
			return 2

		default:
			return 0
		}
	}
	return 0
}

// UsingSPDY indicates whether a given ResponseWriter is using SPDY.
func UsingSPDY(w http.ResponseWriter) bool {
	_, ok := w.(Stream)
	return ok
}
