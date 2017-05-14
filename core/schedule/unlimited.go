package schedule

import (
	"context"

	"github.com/yandex/pandora/core"
)

type unlimited struct {
	base
}

var _ core.Schedule = (*unlimited)(nil)

func (ul *unlimited) Start(ctx context.Context) error {
	defer close(ul.control)
loop:
	for {
		select {
		case ul.control <- struct{}{}:
		case <-ctx.Done():
			break loop

		}
	}
	return nil
}

func NewUnlimited() *unlimited {
	return &unlimited{base: *newBase(64)}
}
