package aggregator

import (
	"context"

	"github.com/yandex/pandora/core"
	"go.uber.org/zap"
)

func NewLog() core.Aggregator {
	return &logging{sink: make(chan core.Sample, 128)}
}

type logging struct {
	sink chan core.Sample
	log  *zap.SugaredLogger
}

func (l *logging) Report(sample core.Sample) {
	l.sink <- sample
}

func (l *logging) Run(ctx context.Context, deps core.AggregatorDeps) error {
	l.log = deps.Log.Sugar()
loop:
	for {
		select {
		case sample := <-l.sink:
			l.handle(sample)
		case <-ctx.Done():
			break loop
		}
	}
	for {
		// Context is done, but we should read all data from sink.
		select {
		case r := <-l.sink:
			l.handle(r)
		default:
			return nil
		}
	}
}

func (l *logging) handle(sample core.Sample) {
	l.log.Info("Sample reported: ", sample)
}
