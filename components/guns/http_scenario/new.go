package httpscenario

import (
	"errors"
	"net"

	phttp "github.com/yandex/pandora/components/guns/http"
	"go.uber.org/zap"
)

func NewHTTPGun(conf phttp.HTTPGunConfig, answLog *zap.Logger, targetResolved string) *BaseGun {
	return newHTTPGun(phttp.HTTP1ClientConstructor, conf, answLog, targetResolved)
}

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(conf phttp.HTTPGunConfig, answLog *zap.Logger, targetResolved string) (*BaseGun, error) {
	if !conf.SSL {
		// Open issue on github if you really need this feature.
		return nil, errors.New("HTTP/2.0 over TCP is not supported. Please leave SSL option true by default")
	}
	return newHTTPGun(phttp.HTTP2ClientConstructor, conf, answLog, targetResolved), nil
}

func newHTTPGun(clientConstructor phttp.ClientConstructor, cfg phttp.HTTPGunConfig, answLog *zap.Logger, targetResolved string) *BaseGun {
	client := clientConstructor(cfg.Client, cfg.Target)
	wrappedClient := phttp.WrapClientHostResolving(client, cfg, targetResolved)
	return &BaseGun{
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
}

func getHostWithoutPort(target string) string {
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		host = target
	}
	return host
}
