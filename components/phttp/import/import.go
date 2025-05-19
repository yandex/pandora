package phttp

import (
	"github.com/spf13/afero"
	phttp "github.com/yandex/pandora/components/guns/http"
	scenarioGun "github.com/yandex/pandora/components/guns/http_scenario"
	httpProvider "github.com/yandex/pandora/components/providers/http"
	scenarioProvider "github.com/yandex/pandora/components/providers/scenario/import"
	"github.com/yandex/pandora/core/register"
)

func Import(fs afero.Fs) {
	httpProvider.Import(fs)
	scenarioGun.Import(fs)
	scenarioProvider.Import(fs)

	register.Gun("http", phttp.NewHTTP1GunFactory, phttp.DefaultHTTPGunConfig)
	register.Gun("http2", phttp.NewHTTP2GunFactory, phttp.DefaultHTTP2GunConfig)
	register.Gun("connect", phttp.NewConnectGunFactory, phttp.DefaultConnectGunConfig)
}
