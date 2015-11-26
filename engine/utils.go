package engine

import (
	"fmt"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/extpoints"
	"github.com/yandex/pandora/gun"
	"github.com/yandex/pandora/limiter"
)

func GetAmmoProvider(c *config.AmmoProvider) (ammo.Provider, error) {
	if c == nil {
		return nil, nil
	}
	provider := extpoints.AmmoProviders.Lookup(c.AmmoType)
	if provider == nil {
		return nil, fmt.Errorf("Provider for ammo '%s' was not found", c.AmmoType)
	}
	return provider(c)
}

func GetResultListener(c *config.ResultListener) (aggregate.ResultListener, error) {
	if c == nil {
		return nil, nil
	}
	listener := extpoints.ResultListeners.Lookup(c.ListenerType)
	if listener == nil {
		return nil, fmt.Errorf("Result listener '%s' was not found", c.ListenerType)
	}
	return listener(c)
}

func GetLimiter(c *config.Limiter) (limiter.Limiter, error) {
	if c == nil {
		return nil, nil
	}
	l := extpoints.Limiters.Lookup(c.LimiterType)
	if l == nil {
		return nil, fmt.Errorf("Limiter type '%s' was not found", c.LimiterType)
	}
	return l(c)
}

func GetGun(c *config.Gun) (gun.Gun, error) {
	if c == nil {
		return nil, nil
	}
	l := extpoints.Guns.Lookup(c.GunType)
	if l == nil {
		return nil, fmt.Errorf("Gun type '%s' was not found", c.GunType)
	}
	return l(c)
}
