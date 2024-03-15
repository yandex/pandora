package phttp

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type HTTPGunConfig struct {
	Base   BaseGunConfig `config:",squash"`
	Client ClientConfig  `config:",squash"`
	Target string        `validate:"endpoint,required"`
	SSL    bool
}

func NewHTTP1Gun(cfg HTTPGunConfig, answLog *zap.Logger, targetResolved string) *BaseGun {
	var wrappedConstructor = func(clientConfig ClientConfig, target string) Client {
		return WrapClientHostResolving(
			HTTP1ClientConstructor(cfg.Client, cfg.Target),
			cfg,
			targetResolved,
		)
	}
	return NewBaseGun(wrappedConstructor, cfg, answLog)
}

func HTTP1ClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewTransport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := NewRedirectingClient(transport, clientConfig.Redirect)
	return client
}

var _ ClientConstructor = HTTP1ClientConstructor

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(cfg HTTPGunConfig, answLog *zap.Logger, targetResolved string) (*BaseGun, error) {
	if !cfg.SSL {
		// Open issue on github if you really need this feature.
		return nil, errors.New("HTTP/2.0 over TCP is not supported. Please leave SSL option true by default.")
	}
	var wrappedConstructor = func(clientConfig ClientConfig, target string) Client {
		return WrapClientHostResolving(
			HTTP2ClientConstructor(cfg.Client, cfg.Target),
			cfg,
			targetResolved,
		)
	}
	return NewBaseGun(wrappedConstructor, cfg, answLog), nil
}

func HTTP2ClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewHTTP2Transport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := NewRedirectingClient(transport, clientConfig.Redirect)
	// Will panic and cancel shooting whet target doesn't support HTTP/2.
	return &panicOnHTTP1Client{Client: client}
}

var _ ClientConstructor = HTTP2ClientConstructor

func DefaultHTTPGunConfig() HTTPGunConfig {
	return HTTPGunConfig{
		SSL:    false,
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
