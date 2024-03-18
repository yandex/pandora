package netsample

import (
	"context"

	"github.com/yandex/pandora/core"
)

type TestAggregator struct {
	Samples []*Sample
}

var _ Aggregator = &TestAggregator{}

func (t *TestAggregator) Run(ctx context.Context, _ core.AggregatorDeps) error {
	<-ctx.Done()
	return nil
}

func (t *TestAggregator) Report(s *Sample) {
	t.Samples = append(t.Samples, s)
}
