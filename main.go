package main

import (
	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/gun"
	"github.com/yandex/pandora/gun/http"
	"github.com/yandex/pandora/gun/spdy"
	"github.com/yandex/pandora/limiter"
	"github.com/yandex/pandora/register"
)

func init() {
	// TODO: make and register NewDefaultConfig funcs

	register.ResultListener("log/simple", aggregate.NewLoggingResultListener)
	register.ResultListener("log/phout", aggregate.GetPhoutResultListener)

	register.Provider("jsonline/http", ammo.NewHttpProvider)
	register.Provider("jsonline/spdy", ammo.NewHttpProvider)
	register.Provider("dummy/log", ammo.NewLogAmmoProvider)

	register.Gun("http", http.New)
	register.Gun("spdy", spdy.New)
	register.Gun("log", gun.NewLog)

	register.Limiter("periodic", limiter.NewPeriodic)
	register.Limiter("composite", limiter.NewComposite)
	register.Limiter("unlimited", limiter.NewUnlimited)
	register.Limiter("linear", limiter.NewLinear)
}

func main() {
	cli.Run()
}
