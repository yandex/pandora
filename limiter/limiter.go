package limiter

import (
	"golang.org/x/net/context"
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
			if !more {
				break loop
			}
			i++
		case <-ctx.Done():
			return i, ctx.Err()
		}
	}
	return i, nil
}
