package example

import (
	"context"
	"log"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregate"
)

func NewGun() *Gun {
	return &Gun{}
}

type Gun struct {
	results core.Results
}

func (l *Gun) BindResultsTo(results core.Results) {
	l.results = results
}

func (l *Gun) Shoot(ctx context.Context, a core.Ammo) error {
	log.Println("logGun message: ", a.(*Ammo).Message)
	l.results <- aggregate.AcquireSample("REQUEST")
	return nil
}
