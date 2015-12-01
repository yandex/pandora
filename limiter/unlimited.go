package limiter

import (
	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

type unlimited struct {
	limiter
}

var _ Limiter = (*unlimited)(nil)

func (pl *unlimited) Start(ctx context.Context) error {
	return nil
}

func NewUnlimitedFromConfig(c *config.Limiter) (l Limiter, err error) {
	// just return nil to show that there are no limits
	return &unlimited{limiter: limiter{}}, nil
}
