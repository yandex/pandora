package phttp

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type GunConfig struct {
	Client         ClientConfig `config:",squash"`
	Target         string       `validate:"endpoint,required"`
	TargetResolved string       `config:"-"`
	SSL            bool

	AutoTag      AutoTagConfig   `config:"auto-tag"`
	AnswLog      AnswLogConfig   `config:"answlog"`
	HTTPTrace    HTTPTraceConfig `config:"httptrace"`
	SharedClient struct {
		ClientNumber int  `config:"client-number,omitempty"`
		Enabled      bool `config:"enabled"`
	} `config:"shared-client,omitempty"`
}

func NewHTTP1Gun(cfg GunConfig, answLog *zap.Logger) *BaseGun {
	return NewBaseGun(HTTP1ClientConstructor, cfg, answLog)
}

func HTTP1ClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewTransport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := NewRedirectingClient(transport, clientConfig.Redirect)
	return client
}

var _ ClientConstructor = HTTP1ClientConstructor

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(cfg GunConfig, answLog *zap.Logger) (*BaseGun, error) {
	if !cfg.SSL {
		// Open issue on github if you really need this feature.
		return nil, errors.New("HTTP/2.0 over TCP is not supported. Please leave SSL option true by default.")
	}
	return NewBaseGun(HTTP2ClientConstructor, cfg, answLog), nil
}

func HTTP2ClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewHTTP2Transport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := NewRedirectingClient(transport, clientConfig.Redirect)
	// Will panic and cancel shooting whet target doesn't support HTTP/2.
	return &panicOnHTTP1Client{Client: client}
}

var _ ClientConstructor = HTTP2ClientConstructor

func DefaultHTTPGunConfig() GunConfig {
	return GunConfig{
		SSL:    false,
		Client: DefaultClientConfig(),
		AutoTag: AutoTagConfig{
			Enabled:     false,
			URIElements: 2,
			NoTagOnly:   true,
		},
		AnswLog: AnswLogConfig{
			Enabled: false,
			Path:    "answ.log",
			Filter:  "error",
		},
		HTTPTrace: HTTPTraceConfig{
			DumpEnabled:  false,
			TraceEnabled: false,
		},
	}
}

func DefaultHTTP2GunConfig() GunConfig {
	return GunConfig{
		Client: DefaultClientConfig(),
		AutoTag: AutoTagConfig{
			Enabled:     false,
			URIElements: 2,
			NoTagOnly:   true,
		},
		AnswLog: AnswLogConfig{
			Enabled: false,
			Path:    "answ.log",
			Filter:  "error",
		},
		HTTPTrace: HTTPTraceConfig{
			DumpEnabled:  false,
			TraceEnabled: false,
		},
		SSL: true,
	}
}
