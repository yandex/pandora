package gun

import (
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
)

type LogGun struct {
	results chan<- *aggregate.Sample
}

func (l *LogGun) BindResultsTo(results chan<- *aggregate.Sample) {
	l.results = results
}

func (l *LogGun) Shoot(ctx context.Context, a ammo.Ammo) error {
	log.Println("Log message: ", a.(*ammo.Log).Message)
	l.results <- aggregate.AcquireSample(float64(time.Now().UnixNano())/1e9, "REQUEST")
	return nil
}

func NewLogGunFromConfig(c *config.Gun) (g Gun, err error) {
	return &LogGun{}, nil
}
