package phttp

import (
	"github.com/spf13/afero"
	phttp "github.com/yandex/pandora/components/guns/http"
	scenarioGun "github.com/yandex/pandora/components/guns/http_scenario"
	httpProvider "github.com/yandex/pandora/components/providers/http"
	scenarioProvider "github.com/yandex/pandora/components/providers/scenario/import"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/lib/answlog"
)

func Import(fs afero.Fs) {
	httpProvider.Import(fs)
	scenarioGun.Import(fs)
	scenarioProvider.Import(fs)

	register.Gun("http", func(conf phttp.HTTPGunConfig) func() core.Gun {
		targetResolved, _ := phttp.PreResolveTargetAddr(&conf.Client, conf.Target)
		answLog := answlog.Init(conf.Base.AnswLog.Path, conf.Base.AnswLog.Enabled)
		return func() core.Gun { return phttp.WrapGun(phttp.NewHTTP1Gun(conf, answLog, targetResolved)) }
	}, phttp.DefaultHTTPGunConfig)

	register.Gun("http2", func(conf phttp.HTTPGunConfig) func() (core.Gun, error) {
		targetResolved, _ := phttp.PreResolveTargetAddr(&conf.Client, conf.Target)
		answLog := answlog.Init(conf.Base.AnswLog.Path, conf.Base.AnswLog.Enabled)
		return func() (core.Gun, error) {
			gun, err := phttp.NewHTTP2Gun(conf, answLog, targetResolved)
			return phttp.WrapGun(gun), err
		}
	}, phttp.DefaultHTTP2GunConfig)

	register.Gun("connect", func(conf phttp.HTTPGunConfig) func() core.Gun {
		conf.Target, _ = phttp.PreResolveTargetAddr(&conf.Client, conf.Target)
		answLog := answlog.Init(conf.Base.AnswLog.Path, conf.Base.AnswLog.Enabled)
		return func() core.Gun {
			return phttp.WrapGun(phttp.NewConnectGun(conf, answLog))
		}
	}, phttp.DefaultConnectGunConfig)
}
