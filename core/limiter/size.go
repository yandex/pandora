package limiter

import (
	"context"

	"github.com/yandex/pandora/lib/utils"
)

// NewSize returns size limiter that cuts master limiter by size
// master shouldn't be started.
func NewSize(size int, master Limiter) Limiter {
	return &sizeLimiter{
		base{make(chan struct{})},
		master,
		size,
	}
}

type sizeLimiter struct {
	base
	master Limiter
	size   int
}

var _ Limiter = (*sizeLimiter)(nil)

func (sl *sizeLimiter) Start(ctx context.Context) error {
	defer close(sl.control)

	masterCtx, cancelMaster := context.WithCancel(ctx)
	masterPromise := utils.PromiseCtx(masterCtx, sl.master.Start)

loop:
	for i := 0; i < sl.size; i++ {
		select {
		case v, more := <-sl.master.Control():
			if !more {
				break loop
			}
			select {
			case sl.control <- v:
			case <-ctx.Done():
				break loop
			}

		case <-ctx.Done():
			break loop
		}
	}
	cancelMaster()
	return <-masterPromise
}
