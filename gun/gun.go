package gun

import (
	"context"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
)

type Gun interface {
	Shoot(context.Context, ammo.Ammo) error
	BindResultsTo(chan<- *aggregate.Sample)
}
