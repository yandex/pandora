package aggregate

import "golang.org/x/net/context"

type ResultListener interface {
	Start(context.Context) error
	Sink() chan<- interface{}
}

type resultListener struct {
	sink chan<- interface{}
}

func (rl *resultListener) Sink() chan<- interface{} {
	return rl.sink
}

func Drain(ctx context.Context, results <-chan interface{}) []interface{} {
	samples := []interface{}{}
loop:
	for {
		select {
		case a, more := <-results:
			if !more {
				break loop
			}
			samples = append(samples, a)
		case <-ctx.Done():
			break loop
		}
	}
	return samples
}
