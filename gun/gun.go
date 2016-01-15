package gun

import (
	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"golang.org/x/net/context"
)

type Gun interface {
	Shoot(context.Context, ammo.Ammo) error
	BindResultsTo(chan<- *aggregate.Sample)
}
