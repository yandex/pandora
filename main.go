package main

import (
	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/gun"
	"github.com/yandex/pandora/gun/http"
	"github.com/yandex/pandora/gun/spdy"
	"github.com/yandex/pandora/limiter"
)

func init() {
	gun.Register("http", http.New)
	gun.Register("spdy", spdy.New)

	ammo.RegisterProvider("jsonline/http", ammo.NewHttpProvider)
	ammo.RegisterProvider("jsonline/spdy", ammo.NewHttpProvider)
	ammo.RegisterProvider("dummy/log", ammo.NewLogAmmoProvider)

	aggregate.RegisterResultListener("log/simple", aggregate.NewLoggingResultListener)
	aggregate.RegisterResultListener("log/phout", aggregate.GetPhoutResultListener)

	limiter.Register("periodic", limiter.NewPeriodic)
	limiter.Register("composite", limiter.NewComposite)
	limiter.Register("unlimited", limiter.NewUnlimited)
	limiter.Register("linear", limiter.NewLinearFromConfig)
}

func main() {
	cli.Run()
}
