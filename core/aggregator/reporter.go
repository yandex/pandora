package aggregator

import (
	"fmt"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coreutil"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type ReporterConfig struct {
	// SampleQueueSize is number maximum number of unhandled samples.
	// On queue overflow, samples are dropped.
	SampleQueueSize int `config:"sample-queue-size" validate:"min=1"`
}

const (
	samplesPerSecondUpperBound             = 128 * 1024
	diskWriteLatencySecondUpperBound       = 0.5
	samplesInQueueAfterDiskWriteUpperBound = samplesPerSecondUpperBound * diskWriteLatencySecondUpperBound
	DefaultSampleQueueSize                 = 2 * samplesInQueueAfterDiskWriteUpperBound
)

func DefaultReporterConfig() ReporterConfig {
	return ReporterConfig{
		SampleQueueSize: DefaultSampleQueueSize,
	}
}

func NewReporter(conf ReporterConfig) *Reporter {
	return &Reporter{
		Incomming: make(chan core.Sample, conf.SampleQueueSize),
	}
}

type Reporter struct {
	Incomming          chan core.Sample
	samplesDropped     atomic.Int64
	lastSampleDropWarn atomic.Int64
}

func (a *Reporter) DroppedErr() error {
	dropped := a.samplesDropped.Load()
	if dropped == 0 {
		return nil
	}
	return &SomeSamplesDropped{dropped}
}

func (a *Reporter) Report(s core.Sample) {
	select {
	case a.Incomming <- s:
	default:
		a.dropSample(s)
	}
}

func (a *Reporter) dropSample(s core.Sample) {
	dropped := a.samplesDropped.Inc()
	if dropped == 1 {
		// AggregatorDeps may not be passed, because Run was not called.
		zap.L().Warn("First sample is dropped. More information in Run error")
	}
	coreutil.ReturnSampleIfBorrowed(s)
}

type SomeSamplesDropped struct {
	Dropped int64
}

func (err *SomeSamplesDropped) Error() string {
	return fmt.Sprintf("%v samples were dropped", err.Dropped)
}
