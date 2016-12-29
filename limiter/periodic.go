package limiter

import (
	"context"
	"time"
)

// periodic implements Limiter interface
type periodic struct {
	base
	ticker *time.Ticker
}

var _ Limiter = (*periodic)(nil)

func (pl *periodic) Start(ctx context.Context) error {
	defer close(pl.control)
	defer pl.ticker.Stop() // don't forget to stop ticker (goroutine leak possible)
	// first tick just after the start
	select {
	case pl.control <- struct{}{}:
	case <-ctx.Done():
		return nil
	}
loop:
	for {
		select {
		case <-pl.ticker.C:
			select {
			case pl.control <- struct{}{}:
			case <-ctx.Done():
				break loop

			}
		case <-ctx.Done():
			break loop
		}
	}
	return nil
}

func newPeriodic(period time.Duration) Limiter {
	return &periodic{
		// timer-based limiters should have big enough cache
		base:   base{make(chan struct{}, 65536)},
		ticker: time.NewTicker(period),
	}
}

// TODO: validate tags
type PeriodicConfig struct {
	Period    time.Duration
	BatchSize int
	MaxCount  int
}

// NewPeriodic returns periodic limiter
func NewPeriodic(conf PeriodicConfig) Limiter {
	l := newPeriodic(conf.Period)
	if conf.MaxCount > 0 {
		l = NewSize(conf.MaxCount, l)
	}
	if conf.BatchSize > 0 {
		l = NewBatch(conf.BatchSize, l)
	}
	return l
}
