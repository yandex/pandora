// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"

	"go.uber.org/zap"
	"golang.org/x/net/http2"

	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/lib/netutil"
	"github.com/pkg/errors"
)

//go:generate mockery -name=Client -case=underscore -inpkg -testonly

type Client interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections() // We should close idle conns after gun close.
}

type ClientConfig struct {
	Redirect  bool            // When true, follow HTTP redirects.
	Dialer    DialerConfig    `config:"dial"`
	Transport TransportConfig `config:",squash"`
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		Transport: DefaultTransportConfig(),
		Dialer:    DefaultDialerConfig(),
		Redirect:  false,
	}
}

// DialerConfig can be mapped on net.Dialer.
// Set net.Dialer for details.
type DialerConfig struct {
	DNSCache bool `config:"dns-cache" map:"-"`

	Timeout   time.Duration `config:"timeout"`
	DualStack bool          `config:"dual-stack"`

	// IPv4/IPv6 settings should not matter really,
	// because target should be dialed using pre-resolved addr.
	FallbackDelay time.Duration `config:"fallback-delay"`
	KeepAlive     time.Duration `config:"keep-alive"`
}

func DefaultDialerConfig() DialerConfig {
	return DialerConfig{
		DNSCache:  true,
		DualStack: true,
		Timeout:   3 * time.Second,
		KeepAlive: 120 * time.Second,
	}
}

func NewDialer(conf DialerConfig) netutil.Dialer {
	d := &net.Dialer{}
	config.Map(d, conf)
	if !conf.DNSCache {
		return d
	}
	return netutil.NewDNSCachingDialer(d, netutil.DefaultDNSCache)
}

// TransportConfig can be mapped on http.Transport.
// See http.Transport for details.
type TransportConfig struct {
	TLSHandshakeTimeout   time.Duration `config:"tls-handshake-timeout"`
	DisableKeepAlives     bool          `config:"disable-keep-alives"`
	DisableCompression    bool          `config:"disable-compression"`
	MaxIdleConns          int           `config:"max-idle-conns"`
	MaxIdleConnsPerHost   int           `config:"max-idle-conns-per-host"`
	IdleConnTimeout       time.Duration `config:"idle-conn-timeout"`
	ResponseHeaderTimeout time.Duration `config:"response-header-timeout"`
	ExpectContinueTimeout time.Duration `config:"expect-continue-timeout"`
}

func DefaultTransportConfig() TransportConfig {
	return TransportConfig{
		MaxIdleConns:          0, // No limit.
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   1 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
	}
}

func NewTransport(conf TransportConfig, dial netutil.DialerFunc) *http.Transport {
	tr := &http.Transport{}
	tr.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,                 // We should not spend time for this stuff.
		NextProtos:         []string{"http/1.1"}, // Disable HTTP/2. Use HTTP/2 transport explicitly, if needed.
	}
	config.Map(tr, conf)
	tr.DialContext = dial
	return tr
}

func NewHTTP2Transport(conf TransportConfig, dial netutil.DialerFunc) *http.Transport {
	tr := NewTransport(conf, dial)
	err := http2.ConfigureTransport(tr)
	if err != nil {
		zap.L().Panic("HTTP/2 transport configure fail", zap.Error(err))
	}
	tr.TLSClientConfig.NextProtos = []string{"h2"}
	return tr
}

func newClient(tr *http.Transport, redirect bool) Client {
	if redirect {
		return redirectClient{&http.Client{Transport: tr}}
	}
	return noRedirectClient{tr}
}

type redirectClient struct{ *http.Client }

func (c redirectClient) CloseIdleConnections() {
	c.Transport.(*http.Transport).CloseIdleConnections()
}

type noRedirectClient struct{ *http.Transport }

func (c noRedirectClient) Do(req *http.Request) (*http.Response, error) {
	return c.Transport.RoundTrip(req)
}

// Used to cancel shooting in HTTP/2 gun, when target doesn't support HTTP/2
type panicOnHTTP1Client struct {
	Client
}

const notHTTP2PanicMsg = "Non HTTP/2 connection established. Seems that target doesn't support HTTP/2."

func (c *panicOnHTTP1Client) Do(req *http.Request) (*http.Response, error) {
	res, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	err = checkHTTP2(res.TLS)
	if err != nil {
		zap.L().Panic(notHTTP2PanicMsg, zap.Error(err))
	}
	return res, nil
}

func checkHTTP2(state *tls.ConnectionState) error {
	if state == nil {
		return errors.New("http2: non TLS connection")
	}
	if p := state.NegotiatedProtocol; p != http2.NextProtoTLS {
		return errors.Errorf("http2: unexpected ALPN protocol %q; want %q", p, http2.NextProtoTLS)
	}
	if !state.NegotiatedProtocolIsMutual {
		return errors.New("http2: could not negotiate protocol mutually")
	}
	return nil
}
