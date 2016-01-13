package gun

import (
	"fmt"
	"log"
	"strconv"

	"golang.org/x/net/context"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
)

type LogGun struct{}

func (l *LogGun) Shoot(ctx context.Context, a ammo.Ammo, results chan<- interface{}) error {
	log.Println("Log message: ", a.(*ammo.Log).Message)
	results <- &aggregate.Sample{}
	return nil
}

type DummySample struct {
	value int
}

func (ds *DummySample) String() string {
	return fmt.Sprintf("My value is %d", ds.value)
}

func (ds *DummySample) AppendToPhout(dst []byte) []byte {
	dst = append(dst, "My value is "...)
	dst = strconv.AppendInt(dst, int64(ds.value), 10)
	dst = append(dst, '\n')
	return dst
}

func NewLogGunFromConfig(c *config.Gun) (g Gun, err error) {
	return &LogGun{}, nil
}
