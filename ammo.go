package main

import (
	"errors"
	"fmt"
)

type AmmoProvider interface {
	Start()
	Source() chan Ammo
}

type ammoProvider struct {
	decoder AmmoDecoder
	source  chan Ammo
}

func (ap *ammoProvider) Source() (s chan Ammo) {
	return ap.source
}

type AmmoDecoder interface {
	FromString(string) (Ammo, error)
}

type Ammo interface {
}

func NewAmmoProviderFromConfig(c *AmmoProviderConfig) (ap AmmoProvider, err error) {
	if c == nil {
		return
	}
	switch c.AmmoType {
	case "jsonline/http":
		ap, err = NewHttpAmmoProvider(c.AmmoSource)
	default:
		err = errors.New(fmt.Sprintf("No such limiter type: %s", c.AmmoType))
	}
	return
}
