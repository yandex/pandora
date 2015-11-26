package extpoints

import (
	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/gun"
	"github.com/yandex/pandora/limiter"
)

type AmmoProvider func(*config.AmmoProvider) (ammo.Provider, error)

type ResultListener func(*config.ResultListener) (aggregate.ResultListener, error)

type Limiter func(*config.Limiter) (limiter.Limiter, error)

type Gun func(*config.Gun) (gun.Gun, error)
