package limiter

import (
	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

type unlimited struct {
	limiter
}

var _ Limiter = (*unlimited)(nil)

func (ul *unlimited) Start(ctx context.Context) error {
	defer close(ul.control)
loop:
	for {
		select {
		case ul.control <- true:
		case <-ctx.Done():
			break loop

		}
	}
	return nil
}

func NewUnlimitedFromConfig(c *config.Limiter) (l Limiter, err error) {
	return &unlimited{limiter: limiter{make(chan bool, 1)}}, nil
}
