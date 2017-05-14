package schedule

import (
	"context"
	"log"
	"time"

	"github.com/yandex/pandora/core"
)

// limiter helps to build more complex limiters
type base struct {
	control chan struct{}
}

func newBase(buf int) *base {
	return &base{make(chan struct{}, buf)}
}

func (l *base) Control() <-chan struct{} {
	return l.control
}

func (l *base) Start(context.Context) error {
	return nil
}

// Drain counts all ticks from limiter
func Drain(ctx context.Context, sch core.Schedule) (int, error) {
	const timeFormat = "2006-01-02T15:04:05.999999"
	i := 0
loop:
	for {
		select {
		case _, more := <-sch.Control():
			log.Printf("Tick: %s", time.Now().Format(timeFormat))
			if !more {
				log.Printf("Exit drain at: %s", time.Now().Format(timeFormat))
				break loop
			}
			i++
		case <-ctx.Done():
			return i, ctx.Err()
		}
	}
	return i, nil
}
