package gun

import (
	"fmt"
	"log"

	"golang.org/x/net/context"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
)

type LogGun struct{}

func (l *LogGun) Shoot(ctx context.Context, a ammo.Ammo, results chan<- aggregate.Sample) error {
	log.Println("Log message: ", a.(*ammo.Log).Message)
	results <- &DummySample{0}
	return nil
}

type DummySample struct {
	value int
}

func (ds *DummySample) PhoutSample() *aggregate.PhoutSample {
	return &aggregate.PhoutSample{}
}

func (ds *DummySample) String() string {
	return fmt.Sprintf("My value is %d", ds.value)
}

func NewLogGunFromConfig(c *config.Gun) (g Gun, err error) {
	return &LogGun{}, nil
}
