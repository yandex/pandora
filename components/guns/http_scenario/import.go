package httpscenario

import (
	"github.com/spf13/afero"
	phttp "github.com/yandex/pandora/components/guns/http"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/lib/answlog"
)

func WrapGun(g Gun) core.Gun {
	if g == nil {
		return nil
	}
	return &gunWrapper{Gun: g}
}

type gunWrapper struct {
	Gun Gun
}

func (g *gunWrapper) Shoot(ammo core.Ammo) {
	g.Gun.Shoot(ammo.(*Scenario))
}

func (g *gunWrapper) Bind(a core.Aggregator, deps core.GunDeps) error {
	return g.Gun.Bind(netsample.UnwrapAggregator(a), deps)
}

func Import(fs afero.Fs) {
	register.Gun("http/scenario", func(conf phttp.HTTPGunConfig) func() core.Gun {
		targetResolved, _ := phttp.PreResolveTargetAddr(&conf.Client, conf.Target)
		answLog := answlog.Init(conf.Base.AnswLog.Path, conf.Base.AnswLog.Enabled)
		return func() core.Gun {
			gun := NewHTTPGun(conf, answLog, targetResolved)
			return WrapGun(gun)
		}
	}, phttp.DefaultHTTPGunConfig)

	register.Gun("http2/scenario", func(conf phttp.HTTPGunConfig) func() (core.Gun, error) {
		targetResolved, _ := phttp.PreResolveTargetAddr(&conf.Client, conf.Target)
		answLog := answlog.Init(conf.Base.AnswLog.Path, conf.Base.AnswLog.Enabled)
		return func() (core.Gun, error) {
			gun, err := NewHTTP2Gun(conf, answLog, targetResolved)
			return WrapGun(gun), err
		}
	}, phttp.DefaultHTTP2GunConfig)
}
