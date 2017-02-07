package main

import (
	"github.com/spf13/afero"
	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/ammo/jsonline"
	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/gun"
	"github.com/yandex/pandora/gun/phttp"
	"github.com/yandex/pandora/limiter"
	"github.com/yandex/pandora/register"
)

func init() {
	// TODO move all registrations to different package,
	// TODO: make and register NewDefaultConfig funcs.

	fs := afero.NewReadOnlyFs(afero.NewOsFs())

	register.ResultListener("log/simple", aggregate.NewLoggingResultListener)
	register.ResultListener("log/phout", aggregate.GetPhoutResultListener)

	newJsonlineProvider := func(conf jsonline.Config) ammo.Provider {
		return jsonline.NewProvider(fs, conf)
	}
	register.Provider("jsonline/http", newJsonlineProvider)
	register.Provider("jsonline/spdy", newJsonlineProvider)
	register.Provider("dummy/log", ammo.NewLogProvider)

	register.Gun("http", phttp.NewHTTPGun, phttp.NewDefaultHTTPGunConfig)
	register.Gun("spdy", phttp.NewSPDYGun)
	register.Gun("log", gun.NewLog)

	register.Limiter("periodic", limiter.NewPeriodic)
	register.Limiter("composite", limiter.NewComposite)
	register.Limiter("unlimited", limiter.NewUnlimited)
	register.Limiter("linear", limiter.NewLinear)
}

func main() {
	cli.Run()
}
