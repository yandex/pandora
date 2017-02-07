package phttp

import (
	"net/http"

	"github.com/yandex/pandora/gun"
)

type HTTPGunConfig struct {
	Target       string `validate:"endpoint,required"`
	SSL          bool
	ClientConfig `config:",squash"`
}

func NewDefaultHTTPGunConfig() HTTPGunConfig {
	return HTTPGunConfig{
		SSL:          false,
		ClientConfig: NewDefaultClientConfig(),
	}
}

func NewHTTPGun(conf HTTPGunConfig) *HTTPGun {
	scheme := "http"
	if conf.SSL {
		scheme = "https"
	}
	transport := NewTransport(conf.TransportConfig)
	transport.DialContext = NewDialer(conf.DialerConfig).DialContext
	var g HTTPGun
	g = HTTPGun{
		Base:   Base{Do: g.Do},
		scheme: scheme,
		target: conf.Target,
		client: &http.Client{Transport: transport},
	}
	return &g
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
