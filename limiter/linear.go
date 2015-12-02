package limiter

import (
	"errors"
	"log"
	"math"
	"time"

	"golang.org/x/net/context"
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
		log.Printf("%f since start", time.Since(startTime).Seconds())
		log.Printf("%f ts", ts)
		waitPeriod := ts - time.Since(startTime).Seconds()
		if waitPeriod > 0 {
			log.Printf("Waiting for %s", (time.Duration(waitPeriod*1e9) * time.Nanosecond))
			select {
			case <-time.After(time.Duration(waitPeriod*1e9) * time.Nanosecond):
				log.Printf("tick at %f since start", time.Since(startTime).Seconds())
				select {
				case l.control <- struct{}{}:
				case <-ctx.Done():
					break loop

				}
			case <-ctx.Done():
				break loop
			}
		} else {
			select {
			case l.control <- struct{}{}:
			case <-ctx.Done():
				break loop

			}
		}
	}
	return nil
}

func NewLinear(startRps float64, endRps float64, period time.Duration) (l Limiter) {
	return &linear{
		// timer-based limiters should have big enough cache
		limiter:  limiter{make(chan struct{}, 65536)},
		startRps: startRps,
		endRps:   endRps,
		period:   period,
	}
}
