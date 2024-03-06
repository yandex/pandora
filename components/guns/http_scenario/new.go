package httpscenario

import (
	"errors"
	"net"

	phttp "github.com/yandex/pandora/components/guns/http"
	"go.uber.org/zap"
)

func NewHTTPGun(conf phttp.HTTPGunConfig, answLog *zap.Logger, targetResolved string) *BaseGun {
	transport := phttp.NewTransport(conf.Client.Transport, phttp.NewDialer(conf.Client.Dialer).DialContext, conf.Gun.Target)
	client := newClient(transport, conf.Client.Redirect)
	return NewClientGun(client, conf.Gun, answLog, targetResolved)
}

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(conf phttp.HTTP2GunConfig, answLog *zap.Logger, targetResolved string) (*BaseGun, error) {
	if !conf.Gun.SSL {
		// Open issue on github if you really need this feature.
		return nil, errors.New("HTTP/2.0 over TCP is not supported. Please leave SSL option true by default")
	}
	transport := phttp.NewHTTP2Transport(conf.Client.Transport, phttp.NewDialer(conf.Client.Dialer).DialContext, conf.Gun.Target)
	client := newClient(transport, conf.Client.Redirect)
	// Will panic and cancel shooting whet target doesn't support HTTP/2.
	client = &panicOnHTTP1Client{client}
	return NewClientGun(client, conf.Gun, answLog, targetResolved), nil
}

func NewClientGun(client Client, conf phttp.GunConfig, answLog *zap.Logger, targetResolved string) *BaseGun {
	scheme := "http"
	if conf.SSL {
		scheme = "https"
	}
	return &BaseGun{
		Config: conf.Base,
		OnClose: func() error {
			client.CloseIdleConnections()
			return nil
		},
		AnswLog:        answLog,
		scheme:         scheme,
		hostname:       getHostWithoutPort(conf.Target),
		targetResolved: targetResolved,
		client:         client,
	}
}

func getHostWithoutPort(target string) string {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	return host
}
