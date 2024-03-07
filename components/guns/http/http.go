package phttp

import (
	"github.com/pkg/errors"
	"github.com/yandex/pandora/core/warmup"
	"go.uber.org/zap"
)

type HTTPGunConfig struct {
	Base   BaseGunConfig `config:",squash"`
	Client ClientConfig  `config:",squash"`
	Target string        `validate:"endpoint,required"`
	SSL    bool
}

func NewHTTPGun(conf HTTPGunConfig, answLog *zap.Logger, targetResolved string) *BaseGun {
	return NewClientGun(HTTP1ClientConstructor, conf, answLog, targetResolved)
}

var HTTP1ClientConstructor clientConstructor = func(clientConfig ClientConfig, target string) Client {
	transport := NewTransport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := newClient(transport, clientConfig.Redirect)
	return client
}

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(conf HTTPGunConfig, answLog *zap.Logger, targetResolved string) (*BaseGun, error) {
	if !conf.SSL {
		// Open issue on github if you really need this feature.
		return nil, errors.New("HTTP/2.0 over TCP is not supported. Please leave SSL option true by default.")
	}
	return NewClientGun(HTTP2ClientConstructor, conf, answLog, targetResolved), nil
}

var HTTP2ClientConstructor clientConstructor = func(clientConfig ClientConfig, target string) Client {
	transport := NewHTTP2Transport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := newClient(transport, clientConfig.Redirect)
	// Will panic and cancel shooting whet target doesn't support HTTP/2.
	return &panicOnHTTP1Client{Client: client}
}

func NewClientGun(clientConstructor clientConstructor, cfg HTTPGunConfig, answLog *zap.Logger, targetResolved string) *BaseGun {
	scheme := "http"
	if cfg.SSL {
		scheme = "https"
	}
	client := clientConstructor(cfg.Client, cfg.Target)
	wrappedClient := &httpDecoratedClient{
		client:         client,
		hostname:       getHostWithoutPort(cfg.Target),
		targetResolved: targetResolved,
		scheme:         scheme,
	}
	g := BaseGun{
		Config: cfg.Base,
		OnClose: func() error {
			client.CloseIdleConnections()
			return nil
		},
		AnswLog: answLog,

		hostname:       getHostWithoutPort(cfg.Target),
		targetResolved: targetResolved,
		client:         wrappedClient,
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

func DefaultHTTPGunConfig() HTTPGunConfig {
	return HTTPGunConfig{

		SSL: false,

		Base:   DefaultBaseGunConfig(),
		Client: DefaultClientConfig(),
	}
}

func DefaultHTTP2GunConfig() HTTPGunConfig {
	return HTTPGunConfig{
		Client: DefaultClientConfig(),
		Base:   DefaultBaseGunConfig(),
		SSL:    true,
	}
}
