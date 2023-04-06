// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.

package phttp

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"go.uber.org/zap"
)

var tunnelHandler = func(t *testing.T, originURL string) http.Handler {
	u, err := url.Parse(originURL)
	require.NoError(t, err)
	originHost := u.Host
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, originHost, r.RequestURI)

		toOrigin, err := net.Dial("tcp", originHost)
		require.NoError(t, err)

		conn, bufReader, err := w.(http.Hijacker).Hijack()
		require.NoError(t, err)
		require.Equal(t, bufReader.Reader.Buffered(), 0, "Current implementation should not send requested data before got response.")

		_, err = io.WriteString(conn, "HTTP/1.1 200 Connection established\r\n\r\n")
		require.NoError(t, err)

		go func() { _, _ = io.Copy(toOrigin, conn) }()
		go func() { _, _ = io.Copy(conn, toOrigin) }()
	})
}

func TestDo(t *testing.T) {
	tests := []struct {
		name      string
		tunnelSSL bool
	}{
		{
			name:      "HTTP client",
			tunnelSSL: false,
		},
		{
			name:      "HTTPS client",
			tunnelSSL: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tunnelSSL := tt.tunnelSSL
			origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
				rw.WriteHeader(http.StatusOK)
			}))
			defer origin.Close()

			var proxy *httptest.Server
			if tunnelSSL {
				proxy = httptest.NewTLSServer(tunnelHandler(t, origin.URL))
			} else {
				proxy = httptest.NewServer(tunnelHandler(t, origin.URL))
			}
			defer proxy.Close()

			req, err := http.NewRequest("GET", origin.URL, nil)
			require.NoError(t, err)

			conf := DefaultConnectGunConfig()
			conf.ConnectSSL = tunnelSSL
			scheme := "http://"
			if tunnelSSL {
				scheme = "https://"
			}
			conf.Target = strings.TrimPrefix(proxy.URL, scheme)

			client := newConnectClient(conf)

			res, err := client.Do(req)
			require.NoError(t, err)
			require.Equal(t, res.StatusCode, http.StatusOK)
		})
	}
}

func TestNewConnectGun(t *testing.T) {

	origin := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}))
	defer origin.Close()
	proxy := httptest.NewServer(tunnelHandler(t, origin.URL))
	defer proxy.Close()

	log := zap.NewNop()
	conf := DefaultConnectGunConfig()
	conf.Target = proxy.Listener.Addr().String()
	connectGun := NewConnectGun(conf, log)

	results := &netsample.TestAggregator{}
	_ = connectGun.Bind(results, testDeps())

	connectGun.Shoot(newAmmoURL(t, origin.URL))

	require.Equal(t, len(results.Samples), 1)
	require.NoError(t, results.Samples[0].Err())
	require.Equal(t, results.Samples[0].ProtoCode(), http.StatusOK)
}
