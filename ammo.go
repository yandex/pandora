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
	case "jsonline/http", "jsonline/spdy":
		ap, err = NewHttpAmmoProvider(c.AmmoSource)
	case "dummy/log":
		ap, err = NewLogAmmoProvider(15)
	default:
		err = errors.New(fmt.Sprintf("No such ammo type: %s", c.AmmoType))
	}
	return
}
