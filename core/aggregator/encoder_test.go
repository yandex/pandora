// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregator

import (
	"context"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	aggregatemock "github.com/yandex/pandora/core/aggregator/mocks"
	coremock "github.com/yandex/pandora/core/mocks"
	iomock "github.com/yandex/pandora/lib/ioutil2/mocks"
	"github.com/yandex/pandora/lib/testutil"
)

type EncoderAggregatorTester struct {
	t          testutil.TestingT
	wc         *iomock.WriteCloser
	sink       *coremock.DataSink
	enc        *aggregatemock.SampleEncoder
	newEncoder NewSampleEncoder
	conf       EncoderAggregatorConfig
	ctx        context.Context
	cancel     context.CancelFunc
	deps       core.AggregatorDeps

	flushCB func()
}

func (tr *EncoderAggregatorTester) Testee() core.Aggregator {
	return NewEncoderAggregator(tr.newEncoder, tr.conf)
}

func (tr *EncoderAggregatorTester) AssertExpectations() {
	t := tr.t
	tr.enc.AssertExpectations(t)
	tr.wc.AssertExpectations(t)
	tr.sink.AssertExpectations(t)
}

func NewEncoderAggregatorTester(t testutil.TestingT) *EncoderAggregatorTester {
	testutil.ReplaceGlobalLogger()
	tr := &EncoderAggregatorTester{t: t}
	tr.wc = &iomock.WriteCloser{}
	tr.sink = &coremock.DataSink{}
	tr.sink.On("OpenSink").Once().Return(tr.wc, nil)
	tr.enc = &aggregatemock.SampleEncoder{}

	tr.newEncoder = func(w io.Writer, flushCB func()) SampleEncoder {
		assert.Equal(t, tr.wc, w)
		tr.flushCB = flushCB
		return tr.enc
	}
	tr.conf = EncoderAggregatorConfig{
		Sink:           tr.sink,
		FlushInterval:  time.Second,
		ReporterConfig: ReporterConfig{100},
	}
	tr.ctx, tr.cancel = context.WithCancel(context.Background())
	tr.deps = core.AggregatorDeps{zap.L()}
	return tr
}

func TestEncoderAggregator(t *testing.T) {
	tr := NewEncoderAggregatorTester(t)
	runErr := make(chan error, 1)

	testee := tr.Testee()
	go func() {
		runErr <- testee.Run(tr.ctx, tr.deps)
	}()

	for i := 0; i < 10; i++ {
		tr.enc.On("Encode", i).Once().Return(nil)
		testee.Report(i)
	}

	tr.enc.On("Flush").Once().Return(func() error {
		tr.wc.On("Close").Once().Return(nil)
		return nil
	})

	tr.cancel()
	err := <-runErr
	require.NoError(t, err)

	tr.AssertExpectations()

	assert.NotPanics(t, func() {
		testee.Report(100)
	})
}

func TestEncoderAggregator_HandleQueueBeforeFinish(t *testing.T) {
	tr := NewEncoderAggregatorTester(t)
	testee := tr.Testee()

	for i := 0; i < 10; i++ {
		tr.enc.On("Encode", i).Once().Return(nil)
		testee.Report(i)
	}
	tr.enc.On("Flush").Once().Return(func() error {
		tr.wc.On("Close").Once().Return(nil)
		return nil
	})

	tr.cancel()
	err := testee.Run(tr.ctx, tr.deps)
	require.NoError(t, err)

	tr.AssertExpectations()
}

func TestEncoderAggregator_CloseSampleEncoder(t *testing.T) {
	tr := NewEncoderAggregatorTester(t)
	newWOCloseEncoder := tr.newEncoder
	tr.newEncoder = func(w io.Writer, onFlush func()) SampleEncoder {
		encoder := newWOCloseEncoder(w, onFlush).(*aggregatemock.SampleEncoder)
		return MockSampleEncoderAddCloser{encoder}
	}
	testee := tr.Testee()

	tr.enc.On("Encode", 0).Once().Return(nil)
	testee.Report(0)

	tr.enc.On("Close").Once().Return(func() error {
		tr.wc.On("Close").Once().Return(nil)
		return nil
	})

	tr.cancel()
	err := testee.Run(tr.ctx, tr.deps)
	require.NoError(t, err)
	tr.AssertExpectations()
}

func TestEncoderAggregator_EverythingFailed(t *testing.T) {
	tr := NewEncoderAggregatorTester(t)
	tr.conf.ReporterConfig.SampleQueueSize = 1
	testee := tr.Testee()

	var (
		encodeErr  = fmt.Errorf("encode")
		flushErr   = fmt.Errorf("flush")
		wcCloseErr = fmt.Errorf("wc close")
	)
	tr.enc.On("Encode", 0).Once().Return(encodeErr)
	testee.Report(0)
	testee.Report(1) // Dropped

	tr.enc.On("Flush").Once().Return(func() error {
		tr.wc.On("Close").Once().Return(wcCloseErr)
		return flushErr
	})

	tr.cancel()
	err := testee.Run(tr.ctx, tr.deps)
	require.Error(t, err)

	wrappedErrors := err.(*multierror.Error).WrappedErrors()
	var causes []error
	for _, err := range wrappedErrors {
		causes = append(causes, errors.Cause(err))
	}
	expectedErrors := []error{encodeErr, flushErr, wcCloseErr, &SomeSamplesDropped{1}}
	assert.Equal(t, expectedErrors, causes)

	tr.AssertExpectations()
}

func TestEncoderAggregator_AutoFlush(t *testing.T) {
	testutil.RunFlaky(t, func(t testutil.TestingT) {
		tr := NewEncoderAggregatorTester(t)
		const flushInterval = 20 * time.Millisecond
		tr.conf.FlushInterval = flushInterval
		testee := tr.Testee()

		var flushes int
		const expectedAutoFlushes = 2
		time.AfterFunc(expectedAutoFlushes*flushInterval+flushInterval/2, tr.cancel)
		tr.enc.On("Flush").Return(func() error {
			flushes++
			return nil
		})
		tr.wc.On("Close").Return(nil)
		err := testee.Run(tr.ctx, tr.deps)
		require.NoError(t, err)

		assert.Equal(t, expectedAutoFlushes+1, flushes, "Expeced + one for finish")
	})
}

func TestEncoderAggregator_ManualFlush(t *testing.T) {
	testutil.RunFlaky(t, func(t testutil.TestingT) {
		tr := NewEncoderAggregatorTester(t)
		const autoFlushInterval = 15 * time.Millisecond
		tr.conf.FlushInterval = autoFlushInterval
		testee := tr.Testee()
		var (
			flushes int
		)

		tr.enc.On("Encode", mock.Anything).Return(func(core.Sample) error {
			tr.flushCB()
			return nil
		})
		tr.enc.On("Flush").Return(func() error {
			flushes++
			tr.flushCB()
			return nil
		})
		tr.wc.On("Close").Return(nil)

		time.AfterFunc(autoFlushInterval*3, tr.cancel)

		runErr := make(chan error)
		go func() {
			runErr <- testee.Run(tr.ctx, tr.deps)
		}()
		writeTicker := time.NewTicker(autoFlushInterval / 3)
		defer writeTicker.Stop()
		for {
			select {
			case _ = <-writeTicker.C:
				testee.Report(0)
			case <-tr.ctx.Done():
				return
			}
		}
		err := <-runErr
		require.NoError(t, err)
		assert.Equal(t, 1, flushes)
		tr.AssertExpectations()
	})
}

type MockSampleEncoderAddCloser struct {
	*aggregatemock.SampleEncoder
}

// Close provides a mock function with given fields:
func (_m MockSampleEncoderAddCloser) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
