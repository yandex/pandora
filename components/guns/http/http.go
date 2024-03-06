package phttp

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/yandex/pandora/core/warmup"
	"go.uber.org/zap"
)

type GunConfig struct {
	Target string `validate:"endpoint,required"`
	SSL    bool
	Base   BaseGunConfig `config:",squash"`
}

type HTTPGunConfig struct {
	Gun    GunConfig    `config:",squash"`
	Client ClientConfig `config:",squash"`
}

type HTTP2GunConfig struct {
	Gun    GunConfig    `config:",squash"`
	Client ClientConfig `config:",squash"`
}

func NewHTTPGun(conf HTTPGunConfig, answLog *zap.Logger, targetResolved string) *HTTPGun {
	return NewClientGun(HTTP1ClientConstructor, conf.Client, conf.Gun, answLog, targetResolved)
}

func HTTP1ClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewTransport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := newClient(transport, clientConfig.Redirect)
	return client
}

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(conf HTTP2GunConfig, answLog *zap.Logger, targetResolved string) (*HTTPGun, error) {
	if !conf.Gun.SSL {
		// Open issue on github if you really need this feature.
		return nil, errors.New("HTTP/2.0 over TCP is not supported. Please leave SSL option true by default.")
	}
	return NewClientGun(HTTP2ClientConstructor, conf.Client, conf.Gun, answLog, targetResolved), nil
}

func HTTP2ClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewHTTP2Transport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := newClient(transport, clientConfig.Redirect)
	// Will panic and cancel shooting whet target doesn't support HTTP/2.
	return &panicOnHTTP1Client{Client: client}
}

func NewClientGun(clientConstructor clientConstructor, clientCfg ClientConfig, gunCfg GunConfig, answLog *zap.Logger, targetResolved string) *HTTPGun {
	client := clientConstructor(clientCfg, gunCfg.Target)
	scheme := "http"
	if gunCfg.SSL {
		scheme = "https"
	}
	var g HTTPGun
	g = HTTPGun{
		BaseGun: BaseGun{
			Config: gunCfg.Base,
			Do:     g.Do,
			OnClose: func() error {
				client.CloseIdleConnections()
				return nil
			},
			AnswLog: answLog,

			scheme:         scheme,
			hostname:       getHostWithoutPort(gunCfg.Target),
			targetResolved: targetResolved,
			client:         client,
		},
	}
	return &g
}

type HTTPGun struct {
	BaseGun
}

var _ Gun = (*HTTPGun)(nil)

func (g *HTTPGun) WarmUp(opts *warmup.Options) (any, error) {
	return nil, nil
}

func (g *HTTPGun) Do(req *http.Request) (*http.Response, error) {
	if req.Host == "" {
		req.Host = g.hostname
	}

	req.URL.Host = g.targetResolved
	req.URL.Scheme = g.scheme
	return g.client.Do(req)
}

func DefaultHTTPGunConfig() HTTPGunConfig {
	return HTTPGunConfig{
		Gun:    DefaultClientGunConfig(),
		Client: DefaultClientConfig(),
	}
}

func DefaultHTTP2GunConfig() HTTP2GunConfig {
	conf := HTTP2GunConfig{
		Client: DefaultClientConfig(),
		Gun:    DefaultClientGunConfig(),
	}
	conf.Gun.SSL = true
	return conf
}

func DefaultClientGunConfig() GunConfig {
	return GunConfig{
		SSL:  false,
		Base: DefaultBaseGunConfig(),
	}
}
