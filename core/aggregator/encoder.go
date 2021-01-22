// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregator

import (
	"context"
	"io"
	"time"

	"github.com/pkg/errors"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coreutil"
	"github.com/yandex/pandora/lib/errutil"
)

type NewSampleEncoder func(w io.Writer, onFlush func()) SampleEncoder

//go:generate mockery -name=SampleEncoder -case=underscore -outpkg=aggregatemock

// SampleEncoder is efficient, buffered encoder of samples.
// SampleEncoder MAY support only concrete type of sample.
// MAY also implement SampleEncodeCloser.
type SampleEncoder interface {
	// SampleEncoder SHOULD panic, if passed sample type is not supported.
	Encode(s core.Sample) error
	// Flush flushes internal buffer to wrapped io.Writer.
	Flush() error
	// Optional. Close MUST be called, if io.Closer is implemented.
	// io.Closer
}

//go:generate mockery -name=SampleEncodeCloser -case=underscore -outpkg=aggregatemock

// SampleEncoderCloser is SampleEncoder that REQUIRE Close call to finish encoding.
type SampleEncodeCloser interface {
	SampleEncoder
	io.Closer
}

type EncoderAggregatorConfig struct {
	Sink           core.DataSink  `config:"sink" validate:"required"`
	BufferSize     int            `config:"buffer-size"`
	FlushInterval  time.Duration  `config:"flush-interval"`
	ReporterConfig ReporterConfig `config:",squash"`
}

func DefaultEncoderAggregatorConfig() EncoderAggregatorConfig {
	return EncoderAggregatorConfig{
		FlushInterval:  time.Second,
		ReporterConfig: DefaultReporterConfig(),
	}
}

// NewEncoderAggregator returns aggregator that use SampleEncoder to marshall samples to core.DataSink.
// Handles encoder flushing and sample dropping on queue overflow.
// putSample is optional func, that called on handled sample. Usually returns sample to pool.
func NewEncoderAggregator(
	newEncoder NewSampleEncoder,
	conf EncoderAggregatorConfig,
) core.Aggregator {
	return &dataSinkAggregator{
		Reporter:   *NewReporter(conf.ReporterConfig),
		newEncoder: newEncoder,
		conf:       conf,
	}
}

type dataSinkAggregator struct {
	Reporter
	core.AggregatorDeps

	newEncoder NewSampleEncoder
	conf       EncoderAggregatorConfig
}

func (a *dataSinkAggregator) Run(ctx context.Context, deps core.AggregatorDeps) (err error) {
	a.AggregatorDeps = deps

	sink, err := a.conf.Sink.OpenSink()
	if err != nil {
		return
	}
	defer func() {
		closeErr := sink.Close()
		err = errutil.Join(err, closeErr)
		err = errutil.Join(err, a.DroppedErr())
	}()

	var flushes int
	encoder := a.newEncoder(sink, func() {
		flushes++
	})
	defer func() {
		if encoder, ok := encoder.(io.Closer); ok {
			closeErr := encoder.Close()
			err = errutil.Join(err, errors.WithMessage(closeErr, "encoder close failed"))
			return
		}
		flushErr := encoder.Flush()
		err = errutil.Join(err, errors.WithMessage(flushErr, "final flush failed"))
	}()

	var flushTick <-chan time.Time
	if a.conf.FlushInterval > 0 {
		flushTicker := time.NewTicker(a.conf.FlushInterval)
		flushTick = flushTicker.C
		defer flushTicker.Stop()
	}

	var previousFlushes int
HandleLoop:
	for {
		select {
		case sample := <-a.Incomming:
			err = a.handleSample(encoder, sample)
			if err != nil {
				return
			}
		case <-flushTick:
			if previousFlushes == flushes {
				a.Log.Debug("Flushing")
				err = encoder.Flush()
				if err != nil {
					return
				}
			}
			previousFlushes = flushes
		case <-ctx.Done():
			break HandleLoop // Still need to handle all queued samples.
		}
	}

	for {
		select {
		case sample := <-a.Incomming:
			err = a.handleSample(encoder, sample)
			if err != nil {
				return
			}
		default:
			return nil
		}
	}
}

func (a *dataSinkAggregator) handleSample(enc SampleEncoder, sample core.Sample) error {
	err := enc.Encode(sample)
	if err != nil {
		return errors.WithMessage(err, "sample encode failed")
	}
	coreutil.ReturnSampleIfBorrowed(sample)
	return nil
}
