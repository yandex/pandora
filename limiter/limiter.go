package limiter

import (
	"context"
	"log"
	"time"
)

// Limiter interface describes limiter control structure
type Limiter interface {
	Start(context.Context) error
	Control() <-chan struct{}
}

// limiter helps to build more complex limiters
type limiter struct {
	control chan struct{}
}

func (l *limiter) Control() <-chan struct{} {
	return l.control
}

func (l *limiter) Start(context.Context) error {
	return nil
}

// Drain counts all ticks from limiter
func Drain(ctx context.Context, l Limiter) (int, error) {
	i := 0
loop:
	for {
		select {
		case _, more := <-l.Control():
			log.Printf("Tick: %s", time.Now().Format("2006-01-02T15:04:05.999999"))
			if !more {
				log.Printf("Exit drain at: %s", time.Now().Format("2006-01-02T15:04:05.999999"))
				break loop
			}
			i++
		case <-ctx.Done():
			return i, ctx.Err()
		}
	}
	return i, nil
}
