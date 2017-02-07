// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"log"
	"net"
	"net/http"
	"time"

	"github.com/yandex/pandora/config"
)

type Client interface {
	Do(r *http.Request) (*http.Response, error)
}

type ClientConfig struct {
	TransportConfig TransportConfig `config:",squash"`
	DialerConfig    DialerConfig    `config:"dial"`
}

func NewDefaultClientConfig() ClientConfig {
	return ClientConfig{
		NewDefaultTransportConfig(),
		NewDefaultDialerConfig(),
	}
}

// DialerConfig can be mapped on net.Dialer.
// Set net.Dialer for details.
type DialerConfig struct {
	Timeout       time.Duration `config:"timeout"`
	DualStack     bool          `config:"dual-stack"`
	FallbackDelay time.Duration `config:"fallback-delay"`
	KeepAlive     time.Duration `config:"keep-alive"`
}

func NewDefaultDialerConfig() DialerConfig {
	return DialerConfig{
		Timeout:   3 * time.Second,
		KeepAlive: 120 * time.Second,
	}
}

func NewDialer(conf DialerConfig) *net.Dialer {
	d := &net.Dialer{}
	err := config.Map(d, conf)
	if err != nil {
		log.Panicf("Dialer config map error: %s", err)
	}
	return d
}

// DialerConfig can be mapped on http.RoundTripper.
// See http.RoundTripper for details.
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

func NewDefaultTransportConfig() TransportConfig {
	return TransportConfig{
		MaxIdleConns:          0, // No limit.
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   1 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}

func NewTransport(conf TransportConfig) *http.Transport {
	tr := &http.Transport{}
	err := config.Map(tr, conf)
	if err != nil {
		log.Panicf("Transport config map error: %s", err)
	}
	return tr
}
