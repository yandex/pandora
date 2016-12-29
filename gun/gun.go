package gun

import (
	"context"
	"reflect"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/plugin"
)

type Gun interface {
	Shoot(context.Context, ammo.Ammo) error
	BindResultsTo(chan<- *aggregate.Sample)
}

func Register(name string, newGun interface{}, newDefaultConfigOptional ...interface{}) {
	plugin.Register(reflect.TypeOf((*Gun)(nil)).Elem(), name, newGun, newDefaultConfigOptional...)
}
