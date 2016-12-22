package limiter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/yandex/pandora/config"
)

// periodic implements Limiter interface
type periodic struct {
	limiter
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

func NewPeriodic(period time.Duration) (l Limiter) {
	return &periodic{
		// timer-based limiters should have big enough cache
		limiter: limiter{make(chan struct{}, 65536)},
		ticker:  time.NewTicker(period),
	}
}

// NewPeriodicFromConfig returns periodic limiter
func NewPeriodicFromConfig(c *config.Limiter) (l Limiter, err error) {
	params := c.Parameters
	if params == nil {
		return nil, errors.New("Parameters not specified")
	}
	period, ok := params["Period"]
	if !ok {
		return nil, errors.New("Period not specified")
	}
	switch t := period.(type) {
	case float64:
		l = NewPeriodic(time.Duration(period.(float64)*1e3) * time.Millisecond)
	default:
		return nil, fmt.Errorf("Period is of the wrong type."+
			" Expected 'float64' got '%T'", t)
	}
	maxCount, ok := params["MaxCount"]
	if ok {
		mc, ok := maxCount.(float64)
		if !ok {
			return nil, fmt.Errorf("MaxCount is of the wrong type."+
				" Expected 'float64' got '%T'", maxCount)
		}
		l = NewSize(int(mc), l)
	}
	batchSize, ok := params["BatchSize"]
	if ok {
		bs, ok := batchSize.(float64)
		if !ok {
			return nil, fmt.Errorf("BatchSize is of the wrong type."+
				" Expected 'float64' got '%T'", batchSize)
		}
		l = NewBatch(int(bs), l)
	}
	return l, nil
}
