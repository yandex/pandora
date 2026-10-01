package phttp

import (
	"fmt"

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
	ResponseCode   string `config:"response-code"`

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

func NewHTTP1GunFactory(conf GunConfig) (func() core.Gun, error) {
	if err := ValidateResponseCode(conf.ResponseCode, false); err != nil {
		return nil, err
	}
	targetResolved, _ := PreResolveTargetAddr(&conf.Client, conf.Target)
	conf.TargetResolved = targetResolved
	return func() core.Gun { return WrapGun(NewHTTP1Gun(conf)) }, nil
}

func HTTP1ClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewTransport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := NewRedirectingClient(transport, clientConfig.Redirect)
	return client
}

var _ ClientConstructor = HTTP1ClientConstructor

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(cfg GunConfig) (*BaseGun, error) {
	if err := ValidateResponseCode(cfg.ResponseCode, true); err != nil {
		return nil, err
	}
	if !cfg.SSL {
		gun := NewBaseGun(H2CClientConstructor, cfg)
		gun.supportsGRPCResponseCode = true
		return gun, nil
	}
	gun := NewBaseGun(HTTP2ClientConstructor, cfg)
	gun.supportsGRPCResponseCode = true
	return gun, nil
}

// ValidateResponseCode checks a shared GunConfig against the selected gun's response capability.
func ValidateResponseCode(mode string, supportsGRPC bool) error {
	switch mode {
	case "", "http":
		return nil
	case "grpc":
		if supportsGRPC {
			return nil
		}
		return fmt.Errorf("response-code %q requires gun type http2", mode)
	default:
		return fmt.Errorf("unsupported response-code %q", mode)
	}
}

func NewHTTP2GunFactory(conf GunConfig) (func() (core.Gun, error), error) {
	if err := ValidateResponseCode(conf.ResponseCode, true); err != nil {
		return nil, err
	}
	targetResolved, _ := PreResolveTargetAddr(&conf.Client, conf.Target)
	conf.TargetResolved = targetResolved
	return func() (core.Gun, error) {
		gun, err := NewHTTP2Gun(conf)
		return WrapGun(gun), err
	}, nil
}

func HTTP2ClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewHTTP2Transport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	client := NewRedirectingClient(transport, clientConfig.Redirect)
	// Will panic and cancel shooting whet target doesn't support HTTP/2.
	return &panicOnHTTP1Client{Client: client}
}

func H2CClientConstructor(clientConfig ClientConfig, target string) Client {
	transport := NewH2CTransport(clientConfig.Transport, NewDialer(clientConfig.Dialer).DialContext, target)
	return NewRedirectingClient(transport, clientConfig.Redirect)
}

var _ ClientConstructor = HTTP2ClientConstructor
var _ ClientConstructor = H2CClientConstructor

func DefaultHTTPGunConfig() GunConfig {
	return GunConfig{
		SSL:          false,
		ResponseCode: "http",
		Client:       DefaultClientConfig(),
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
		Client:       DefaultClientConfig(),
		ResponseCode: "http",
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
