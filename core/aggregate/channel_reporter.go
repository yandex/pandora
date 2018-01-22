// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregate

import (
	"fmt"

	"go.uber.org/atomic"

	"github.com/yandex/pandora/core"
)

type ChannelReporterConfig struct {
	// SampleQueueSize is number maximum number of unhandled samples.
	// On queue overflow, samples are dropped.
	SampleQueueSize int `config:"sample-buff-size"`
}

const (
	samplesPerSecondUpperBound             = 128 * 1024
	diskWriteLatencySecondUpperBound       = 0.5
	samplesInQueueAfterDiskWriteUpperBound = samplesPerSecondUpperBound * diskWriteLatencySecondUpperBound
	DefaultSampleQueueSize                 = 2 * samplesInQueueAfterDiskWriteUpperBound
)

func NewDefaultReporterConfig() ChannelReporterConfig {
	return ChannelReporterConfig{
		SampleQueueSize: DefaultSampleQueueSize,
	}
}

func NewChannelReporter(conf ChannelReporterConfig) *ChannelReporter {
	return &ChannelReporter{
		Incomming: make(chan core.Sample, conf.SampleQueueSize),
	}
}

type ChannelReporter struct {
	core.AggregatorDeps
	Incomming          chan core.Sample
	samplesDropped     atomic.Int64
	lastSampleDropWarn atomic.Int64
}

func (a *ChannelReporter) DroppedErr() error {
	dropped := a.samplesDropped.Load()
	if dropped == 0 {
		return nil
	}
	return &SomeSamplesDropped{dropped}
}

func (a *ChannelReporter) Report(s core.Sample) {
	select {
	case a.Incomming <- a:
	default:
		a.dropSample(s)
	}
}

func (a *ChannelReporter) dropSample(s core.Sample) {
	dropped := a.samplesDropped.Inc()
	if dropped == 1 {
		a.Log.Warn("First sample is dropped. More information in Run error")
	}
	core.ReturnSampleIfBorrowed(s)
}

type SomeSamplesDropped struct {
	Dropped int64
}

func (err *SomeSamplesDropped) Error() string {
	return fmt.Sprintf("%v samples were dropped", err.Dropped)
}
