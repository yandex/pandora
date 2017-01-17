// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy_test

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/SlyMarbo/spdy"
)

func init() {
	// spdy.EnableDebugOutput()
}

func TestProxyConnect(t *testing.T) {
	cert, err := tls.X509KeyPair(localhostCert, localhostKey)
	if err != nil {
		panic(fmt.Sprintf("could not read certificate: %v", err))
	}

	serverTLSConfig := new(tls.Config)
	serverTLSConfig.Certificates = []tls.Certificate{cert}

	conn, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		panic(fmt.Sprintf("could not listen: %v", err))
	}

	listener := tls.NewListener(conn, serverTLSConfig)

	errChan := make(chan error)

	go func() {
		srv := &http.Server{
			Addr: conn.Addr().String(),
			Handler: spdy.ProxyConnections(spdy.ProxyConnHandlerFunc(func(conn spdy.Conn) {
				req, err := http.NewRequest("GET", "http://example.com/", nil)
				if err != nil {
					errChan <- err
					return
				}
				resp, err := conn.RequestResponse(req, nil, 2)
				if err != nil {
					errChan <- err
					return
				}
				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					errChan <- err
					return
				}

				if !bytes.Equal(body, []byte("HELLO")) {
					errChan <- fmt.Errorf("Expected HELLO. Got %v", string(body))
					return
				}

				close(errChan)
			})),
		}
		srv.Serve(listener)
		println("Serve done")
	}()

	clientTLSConfig := &tls.Config{InsecureSkipVerify: true}

	url := "https://" + conn.Addr().String()

	go func() {
		err = spdy.ConnectAndServe(url, clientTLSConfig, &http.Server{
			Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					errChan <- fmt.Errorf("Expected method GET. Got: %v", r.Method)
				}
				if r.URL.String() != "http://example.com/" {
					errChan <- fmt.Errorf("Expected http://example.com. Got %v", r.URL)
				}
				w.Write([]byte("HELLO"))
			}),
		})
		if err != nil {
			errChan <- fmt.Errorf("ConnectAndServeFailed: %v", err)
		}
	}()

	select {
	case err = <-errChan:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(time.Second):
		t.Error("Timeout")
	}
}

// localhostCert is a PEM-encoded TLS cert with SAN IPs
// "127.0.0.1" and "[::1]", expiring at the last second of 2049 (the end
// of ASN.1 time).
// generated from src/pkg/crypto/tls:
// go run generate_cert.go  --rsa-bits 512 --host 127.0.0.1,::1,example.com --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
var localhostCert = []byte(`-----BEGIN CERTIFICATE-----
MIIBdzCCASOgAwIBAgIBADALBgkqhkiG9w0BAQUwEjEQMA4GA1UEChMHQWNtZSBD
bzAeFw03MDAxMDEwMDAwMDBaFw00OTEyMzEyMzU5NTlaMBIxEDAOBgNVBAoTB0Fj
bWUgQ28wWjALBgkqhkiG9w0BAQEDSwAwSAJBAN55NcYKZeInyTuhcCwFMhDHCmwa
IUSdtXdcbItRB/yfXGBhiex00IaLXQnSU+QZPRZWYqeTEbFSgihqi1PUDy8CAwEA
AaNoMGYwDgYDVR0PAQH/BAQDAgCkMBMGA1UdJQQMMAoGCCsGAQUFBwMBMA8GA1Ud
EwEB/wQFMAMBAf8wLgYDVR0RBCcwJYILZXhhbXBsZS5jb22HBH8AAAGHEAAAAAAA
AAAAAAAAAAAAAAEwCwYJKoZIhvcNAQEFA0EAAoQn/ytgqpiLcZu9XKbCJsJcvkgk
Se6AbGXgSlq+ZCEVo0qIwSgeBqmsJxUu7NCSOwVJLYNEBO2DtIxoYVk+MA==
-----END CERTIFICATE-----`)

// localhostKey is the private key for localhostCert.
var localhostKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIBPAIBAAJBAN55NcYKZeInyTuhcCwFMhDHCmwaIUSdtXdcbItRB/yfXGBhiex0
0IaLXQnSU+QZPRZWYqeTEbFSgihqi1PUDy8CAwEAAQJBAQdUx66rfh8sYsgfdcvV
NoafYpnEcB5s4m/vSVe6SU7dCK6eYec9f9wpT353ljhDUHq3EbmE4foNzJngh35d
AekCIQDhRQG5Li0Wj8TM4obOnnXUXf1jRv0UkzE9AHWLG5q3AwIhAPzSjpYUDjVW
MCUXgckTpKCuGwbJk7424Nb8bLzf3kllAiA5mUBgjfr/WtFSJdWcPQ4Zt9KTMNKD
EUO0ukpTwEIl6wIhAMbGqZK3zAAFdq8DD2jPx+UJXnh0rnOkZBzDtJ6/iN69AiEA
1Aq8MJgTaYsDQWyU/hDq5YkDJc9e9DSCvUIzqxQWMQE=
-----END RSA PRIVATE KEY-----`)
