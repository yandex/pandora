package gun

import (
	"context"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
)

type Gun interface {
	Shoot(context.Context, ammo.Ammo) error
	BindResultsTo(Results)
}

type Results chan<- *aggregate.Sample

func NewResults(buf int) chan *aggregate.Sample {
	return make(chan *aggregate.Sample, buf)
}
