package phttp

import (
	"github.com/pkg/errors"
	"github.com/yandex/pandora/components/answ/filter"
	"github.com/yandex/pandora/components/answ/sampler"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/answlog"
)

type GunConfig struct {
	Client         ClientConfig `config:",squash"`
	Target         string       `validate:"endpoint,required"`
	TargetResolved string       `config:"-"`
	SSL            bool

	AutoTag      AutoTagConfig   `config:"auto-tag"`
	AnswLog      answlog.Config  `config:"answlog"`
	HTTPTrace    HTTPTraceConfig `config:"httptrace"`
	SharedClient struct {
		ClientNumber int  `config:"client-number,omitempty"`
		Enabled      bool `config:"enabled"`
	} `config:"shared-client,omitempty"`
}

func NewHTTP1Gun(cfg GunConfig) *BaseGun {
	return NewBaseGun(HTTP1ClientConstructor, cfg)
}

func NewHTTP1GunFactory(conf GunConfig) func() core.Gun {
	targetResolved, _ := PreResolveTargetAddr(&conf.Client, conf.Target)
	conf.TargetResolved = targetResolved
	return func() core.Gun { return WrapGun(NewHTTP1Gun(conf)) }
}

func HTTP1ClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewTransport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := NewRedirectingClient(transport, clientConfig.Redirect)
	return client
}

var _ ClientConstructor = HTTP1ClientConstructor

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(cfg GunConfig) (*BaseGun, error) {
	if !cfg.SSL {
		// Open issue on github if you really need this feature.
		return nil, errors.New("HTTP/2.0 over TCP is not supported. Please leave SSL option true by default.")
	}
	return NewBaseGun(HTTP2ClientConstructor, cfg), nil
}

func NewHTTP2GunFactory(conf GunConfig) func() (core.Gun, error) {
	targetResolved, _ := PreResolveTargetAddr(&conf.Client, conf.Target)
	conf.TargetResolved = targetResolved
	return func() (core.Gun, error) {
		gun, err := NewHTTP2Gun(conf)
		return WrapGun(gun), err
	}
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
		AnswLog: answlog.Config{
			Enabled: false,
			Filter:  filter.FilterAll,
			Sampling: answlog.Sampling{
				Enabled: true,
				Pattern: sampler.FactorPattern{
					Factor: 10,
				},
			},
			Masking: &answlog.PassAllMasker{},
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
		AnswLog: answlog.Config{
			Enabled: false,
			Path:    "answ.log",
			Filter:  filter.FilterAll,
			Sampling: answlog.Sampling{
				Enabled: true,
				Pattern: sampler.FactorPattern{
					Factor: 10,
				},
			},
			Masking: &answlog.PassAllMasker{},
		},
		HTTPTrace: HTTPTraceConfig{
			DumpEnabled:  false,
			TraceEnabled: false,
		},
		SSL: true,
	}
}
