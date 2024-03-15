package httpscenario

import (
	"errors"

	phttp "github.com/yandex/pandora/components/guns/http"
	"go.uber.org/zap"
)

func NewHTTPGun(conf phttp.HTTPGunConfig, answLog *zap.Logger, targetResolved string) *ScenarioGun {
	return newScenarioGun(phttp.HTTP1ClientConstructor, conf, answLog, targetResolved)
}

// NewHTTP2Gun return simple HTTP/2 gun that can shoot sequentially through one connection.
func NewHTTP2Gun(conf phttp.HTTPGunConfig, answLog *zap.Logger, targetResolved string) (*ScenarioGun, error) {
	if !conf.SSL {
		// Open issue on github if you really need this feature.
		return nil, errors.New("HTTP/2.0 over TCP is not supported. Please leave SSL option true by default")
	}
	return newScenarioGun(phttp.HTTP2ClientConstructor, conf, answLog, targetResolved), nil
}

func newScenarioGun(clientConstructor phttp.ClientConstructor, cfg phttp.HTTPGunConfig, answLog *zap.Logger, targetResolved string) *ScenarioGun {
	var wrappedConstructor = func(clientConfig phttp.ClientConfig, target string) phttp.Client {
		return phttp.WrapClientHostResolving(
			clientConstructor(cfg.Client, cfg.Target),
			cfg,
			targetResolved,
		)
	}

	return &ScenarioGun{
		base: phttp.NewBaseGun(wrappedConstructor, cfg, answLog),
	}
}
