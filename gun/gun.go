package gun

import (
	"github.com/yandex/pandora/ammo"
	"golang.org/x/net/context"
)

type Gun interface {
	Shoot(context.Context, ammo.Ammo, chan<- interface{}) error
}
