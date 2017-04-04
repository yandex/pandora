package gun

import (
	"context"
	"log"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
)

type logGun struct {
	results Results
}

func (l *logGun) BindResultsTo(results Results) {
	l.results = results
}

func (l *logGun) Shoot(ctx context.Context, a ammo.Ammo) error {
	log.Println("logGun message: ", a.(*ammo.Log).Message)
	l.results <- aggregate.AcquireSample("REQUEST")
	return nil
}

func NewLog() Gun {
	return &logGun{}
}
