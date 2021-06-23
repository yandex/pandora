// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"bufio"
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/yandex/pandora/lib/netutil"
	"github.com/pkg/errors"
)

type ConnectGunConfig struct {
	Target        string       `validate:"endpoint,required"`
	ConnectSSL    bool         `config:"connect-ssl"` // Defines if tunnel encrypted.
	SSL           bool         // As in HTTP gun, defines scheme for http requests.
	Client        ClientConfig `config:",squash"`
	BaseGunConfig `config:",squash"`
}

func NewConnectGun(conf ConnectGunConfig) *ConnectGun {
	scheme := "http"
	if conf.SSL {
		scheme = "https"
	}
	client := newConnectClient(conf)
	var g ConnectGun
	g = ConnectGun{
		BaseGun: BaseGun{
			Config: conf.BaseGunConfig,
			Do:     g.Do,
			OnClose: func() error {
				client.CloseIdleConnections()
				return nil
			},
		},
		scheme: scheme,
		client: client,
	}
	return &g
}

type ConnectGun struct {
	BaseGun
	scheme string
	client Client
}

var _ Gun = (*ConnectGun)(nil)

func (g *ConnectGun) Do(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = g.scheme
	return g.client.Do(req)
}

func DefaultConnectGunConfig() ConnectGunConfig {
	return ConnectGunConfig{
		SSL:        false,
		ConnectSSL: false,
		Client:     DefaultClientConfig(),
	}
}

func newConnectClient(conf ConnectGunConfig) Client {
	transport := NewTransport(conf.Client.Transport,
		newConnectDialFunc(
			conf.Target,
			conf.ConnectSSL,
			NewDialer(conf.Client.Dialer),
		))
	return newClient(transport, conf.Client.Redirect)
}

func newConnectDialFunc(target string, connectSSL bool, dialer netutil.Dialer) netutil.DialerFunc {
	return func(ctx context.Context, network, address string) (conn net.Conn, err error) {
		// TODO(skipor): make connect sample.
		// TODO(skipor): make httptrace callbacks called correctly.
		if network != "tcp" {
			panic("unsupported network " + network)
		}
		defer func() {
			if err != nil && conn != nil {
				_ = conn.Close()
				conn = nil
			}
		}()
		conn, err = dialer.DialContext(ctx, "tcp", target)
		if err != nil {
			err = errors.WithStack(err)
			return
		}
		if connectSSL {
			conn = tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
		}
		req := &http.Request{
			Method:     "CONNECT",
			URL:        &url.URL{},
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       nil,
			Host:       address,
		}
		// NOTE(skipor): any logic for CONNECT request can be easily added via hooks.
		err = req.Write(conn)
		if err != nil {
			err = errors.WithStack(err)
			return
		}
		// NOTE(skipor): according to RFC 2817 we can send origin at that moment and not wait
		// for request. That requires to wrap conn and do following logic at first read.
		r := bufio.NewReader(conn)
		res, err := http.ReadResponse(r, req)
		if err != nil {
			err = errors.WithStack(err)
			return
		}
		// RFC 7230 3.3.3.2: Any 2xx (Successful) response to a CONNECT request implies that
		// the connection will become a tunnel immediately after the empty
		// line that concludes the header fields. A client MUST ignore any
		// Content-Length or Transfer-Encoding header fields received in
		// such a message.
		if res.StatusCode != http.StatusOK {
			dump, dumpErr := httputil.DumpResponse(res, false)
			err = errors.Errorf("Unexpected status code. Dumped response:\n%s\n Dump error: %s",
				dump, dumpErr)
			return
		}
		// No need to close body.
		if r.Buffered() != 0 {
			// Already receive something non HTTP from proxy or dialed server.
			// Anyway it is incorrect situation.
			peek, _ := r.Peek(r.Buffered())
			err = errors.Errorf("Unexpected extra data after connect: %q", peek)
			return
		}
		return
	}
}
