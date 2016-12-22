package limiter

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/yandex/pandora/config"
)

// linear implements Limiter interface
type linear struct {
	limiter
	startRps float64
	endRps   float64
	period   time.Duration
}

var _ Limiter = (*linear)(nil)

func quadraticRightRoot(a, b, c float64) (float64, error) {
	discriminant := math.Pow(b, 2) - 4*a*c
	if discriminant < 0 {
		return 0, errors.New("Discriminant is less then zero")
	}
	root := (-b + math.Sqrt(discriminant)) / (2 * a)
	return root, nil
}

func (l *linear) Start(ctx context.Context) error {
	defer close(l.control)
	a := (l.endRps - l.startRps) / l.period.Seconds() / 2.0
	b := l.startRps
	maxCount := (a*math.Pow(l.period.Seconds(), 2) + b*l.period.Seconds())
	startTime := time.Now()
loop:
	for n := 0.0; n < maxCount; n += 1.0 {
		ts, err := quadraticRightRoot(a, b, -n)
		if err != nil {
			return err
		}
		waitPeriod := ts - time.Since(startTime).Seconds()
		if waitPeriod > 0 {
			select {
			case <-time.After(time.Duration(waitPeriod*1e9) * time.Nanosecond):
			case <-ctx.Done():
				break loop
			}
		}
		select {
		case l.control <- struct{}{}:
		case <-ctx.Done():
			break loop

		}

	}
	// now wait until the end of specified period
	waitPeriod := l.period.Seconds() - time.Since(startTime).Seconds()
	if waitPeriod > 0 {
		select {
		case <-time.After(time.Duration(waitPeriod*1e9) * time.Nanosecond):
		case <-ctx.Done():
		}
	}
	return nil
}

func NewLinear(startRps, endRps, period float64) (l Limiter) {
	return &linear{
		// timer-based limiters should have big enough cache
		limiter:  limiter{make(chan struct{}, 65536)},
		startRps: startRps,
		endRps:   endRps,
		period:   time.Duration(period*1e9) * time.Nanosecond,
	}
}

// NewLinearFromConfig returns linear limiter
func NewLinearFromConfig(c *config.Limiter) (l Limiter, err error) {
	params := c.Parameters
	if params == nil {
		return nil, errors.New("Parameters not specified")
	}
	period, ok := params["Period"]
	if !ok {
		return nil, errors.New("Period not specified")
	}
	fPeriod, ok := period.(float64)
	if !ok {
		return nil, fmt.Errorf("Period is of the wrong type."+
			" Expected 'float64' got '%T'", period)
	}
	startRps, ok := params["StartRps"]
	if !ok {
		return nil, fmt.Errorf("StartRps should be specified")
	}
	fStartRps, ok := startRps.(float64)
	if !ok {
		return nil, fmt.Errorf("StartRps is of the wrong type."+
			" Expected 'float64' got '%T'", startRps)
	}
	endRps, ok := params["EndRps"]
	if !ok {
		return nil, fmt.Errorf("EndRps should be specified")
	}
	fEndRps, ok := endRps.(float64)
	if !ok {
		return nil, fmt.Errorf("EndRps is of the wrong type."+
			" Expected 'float64' got '%T'", endRps)
	}
	return NewLinear(fStartRps, fEndRps, fPeriod), nil
}
