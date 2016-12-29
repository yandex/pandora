package aggregate

import (
	"context"
	"reflect"

	"github.com/yandex/pandora/plugin"
)

func RegisterResultListener(name string, newResultListener interface{}, newDefaultConfigOptional ...interface{}) {
	plugin.Register(reflect.TypeOf((*ResultListener)(nil)).Elem(), name, newResultListener, newDefaultConfigOptional...)
}

type ResultListener interface {
	Start(context.Context) error
	Sink() chan<- *Sample
}

type resultListener struct {
	sink chan<- *Sample
}

func (rl *resultListener) Sink() chan<- *Sample {
	return rl.sink
}

func Drain(ctx context.Context, results <-chan *Sample) []*Sample {
	samples := []*Sample{}
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
