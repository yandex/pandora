package aggregate

import "golang.org/x/net/context"

type Sample interface {
	String() string
}

type ResultListener interface {
	Start(context.Context) error
	Sink() chan<- Sample
}

func Drain(ctx context.Context, results <-chan Sample) []Sample {
	samples := []Sample{}
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
