package netsample

import (
	"context"

	"github.com/yandex/pandora/core"
)

//go:generate go run github.com/vektra/mockery/v2@v2.22.1 --inpackage --name=Aggregator --filename=mock_aggregator.go

type Aggregator interface {
	Run(ctx context.Context, deps core.AggregatorDeps) error
	Report(sample *Sample)
}

func WrapAggregator(a Aggregator) core.Aggregator { return &aggregatorWrapper{a} }

func UnwrapAggregator(a core.Aggregator) Aggregator {
	switch a := a.(type) {
	case *aggregatorWrapper:
		return a.Aggregator
	}
	return &aggregatorUnwrapper{a}
}

type aggregatorWrapper struct{ Aggregator }

func (a *aggregatorWrapper) Report(s core.Sample) { a.Aggregator.Report(s.(*Sample)) }

type aggregatorUnwrapper struct{ core.Aggregator }

func (a *aggregatorUnwrapper) Report(s *Sample) { a.Aggregator.Report(s) }
