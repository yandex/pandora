package main

import (
	"github.com/spf13/afero"

	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/components/example"
	"github.com/yandex/pandora/components/phttp"
	"github.com/yandex/pandora/components/phttp/ammo/jsonline"
	"github.com/yandex/pandora/components/phttp/ammo/uri"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregate"
	"github.com/yandex/pandora/core/limiter"
	"github.com/yandex/pandora/core/register"
)

func init() {
	// TODO(skipor): move all registrations to different package,
	// TODO(skipor): make and register NewDefaultConfig funcs.

	// TODO(skipor): use afero in result listeners
	register.ResultListener("simple", aggregate.NewLoggingResultListener)
	register.ResultListener("phout", aggregate.GetPhoutResultListener)

	register.Limiter("periodic", limiter.NewPeriodic)
	register.Limiter("composite", limiter.NewComposite)
	register.Limiter("unlimited", limiter.NewUnlimited)
	register.Limiter("linear", limiter.NewLinear)

	fs := afero.NewReadOnlyFs(afero.NewOsFs())

	register.Provider("jsonline", func(conf jsonline.Config) core.Provider {
		return jsonline.NewProvider(fs, conf)
	})
	register.Provider("uri", func(conf uri.Config) core.Provider {
		return uri.NewProvider(fs, conf)
	})

	register.Gun("http", phttp.NewHTTPGunClient, phttp.NewDefaultHTTPGunClientConfig)
	register.Gun("connect", phttp.NewConnectGun, phttp.NewDefaultConnectGunConfig)
	register.Gun("spdy", phttp.NewSPDYGun)

	register.Provider("example", example.NewLogProvider)
	register.Gun("example", example.NewGun)
}

func main() {
	cli.Run()
}
