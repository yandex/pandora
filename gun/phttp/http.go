package phttp

import (
	"net/http"

	"github.com/yandex/pandora/gun"
)

type HTTPGunConfig struct {
	Target string `validate:"endpoint,required"`
	SSL    bool
}

func NewHTTPGun(client Client, conf HTTPGunConfig) *HTTPGun {
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

type HTTPGunClientConfig struct {
	Gun    HTTPGunConfig `config:",squash"`
	Client ClientConfig  `config:",squash"`
}

func NewHTTPGunClient(conf HTTPGunClientConfig) *HTTPGun {
	transport := NewTransport(conf.Client.Transport)
	transport.DialContext = NewDialer(conf.Client.Dialer).DialContext
	client := &http.Client{Transport: transport}
	return NewHTTPGun(client, conf.Gun)
}

type HTTPGun struct {
	Base
	scheme string
	target string
	client Client
}

var _ gun.Gun = (*HTTPGun)(nil)

func (g *HTTPGun) Do(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = g.scheme
	req.URL.Host = g.target
	return g.client.Do(req)
}

func NewDefaultHTTPGunClientConfig() HTTPGunClientConfig {
	return HTTPGunClientConfig{
		Gun:    NewDefaultHTTPGunConfig(),
		Client: NewDefaultClientConfig(),
	}
}

func NewDefaultHTTPGunConfig() HTTPGunConfig {
	return HTTPGunConfig{
		SSL: false,
	}
}
