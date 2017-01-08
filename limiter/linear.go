package limiter

import (
	"context"
	"errors"
	"math"
	"time"
)

func NewLinear(conf LinearConfig) Limiter {
	return &linear{
		// timer-based limiters should have big enough cache
		base:         *newBase(65536),
		LinearConfig: conf,
	}
}

type LinearConfig struct {
	Period   time.Duration
	StartRps float64
	EndRps   float64
}

// linear implements Limiter interface
type linear struct {
	base
	LinearConfig
}

var _ Limiter = (*linear)(nil)

func (l *linear) Start(ctx context.Context) error {
	defer close(l.control)
	a := (l.EndRps - l.StartRps) / l.Period.Seconds() / 2.0
	b := l.StartRps
	maxCount := (a*math.Pow(l.Period.Seconds(), 2) + b*l.Period.Seconds())
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
	waitPeriod := l.Period.Seconds() - time.Since(startTime).Seconds()
	if waitPeriod > 0 {
		select {
		case <-time.After(time.Duration(waitPeriod*1e9) * time.Nanosecond):
		case <-ctx.Done():
		}
	}
	return nil
}

func quadraticRightRoot(a, b, c float64) (float64, error) {
	discriminant := math.Pow(b, 2) - 4*a*c
	if discriminant < 0 {
		return 0, errors.New("Discriminant is less then zero")
	}
	root := (-b + math.Sqrt(discriminant)) / (2 * a)
	return root, nil
}
