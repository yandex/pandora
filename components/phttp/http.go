// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"net/http"
)

type ClientGunConfig struct {
	Target string `validate:"endpoint,required"`
	SSL    bool
}

type HTTPGunConfig struct {
	Gun    ClientGunConfig `config:",squash"`
	Client ClientConfig    `config:",squash"`
}

func NewHTTPGun(conf HTTPGunConfig) *HTTPGun {
	transport := NewTransport(conf.Client.Transport)
	transport.DialContext = NewDialer(conf.Client.Dialer).DialContext
	client := &http.Client{Transport: transport}
	return NewClientGun(client, conf.Gun)
}

func NewClientGun(client Client, conf ClientGunConfig) *HTTPGun {
	scheme := "http"
	if conf.SSL {
		scheme = "https"
	}
	var g HTTPGun
	g = HTTPGun{
		Base:   Base{Do: g.Do},
		scheme: scheme,
		target: conf.Target,
		client: client,
	}
	return &g
}

type HTTPGun struct {
	Base
	scheme string
	target string
	client Client
}

var _ Gun = (*HTTPGun)(nil)

func (g *HTTPGun) Do(req *http.Request) (*http.Response, error) {
	req.Host = req.URL.Host
	req.URL.Host = g.target
	req.URL.Scheme = g.scheme
	return g.client.Do(req)
}

func NewDefaultHTTPGunConfig() HTTPGunConfig {
	return HTTPGunConfig{
		Gun:    NewDefaultClientGunConfig(),
		Client: NewDefaultClientConfig(),
	}
}

func NewDefaultClientGunConfig() ClientGunConfig {
	return ClientGunConfig{
		SSL: false,
	}
}
