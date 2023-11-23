package engine

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator"
	"github.com/yandex/pandora/core/config"
	coremock "github.com/yandex/pandora/core/mocks"
	"github.com/yandex/pandora/core/provider"
	"github.com/yandex/pandora/core/schedule"
	"github.com/yandex/pandora/lib/monitoring"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Test_ConfigValidation(t *testing.T) {
	t.Run("dive validation", func(t *testing.T) {
		conf := Config{
			Pools: []InstancePoolConfig{
				{},
			},
		}
		err := config.Validate(conf)
		require.Error(t, err)
	})
	t.Run("pools required", func(t *testing.T) {
		conf := Config{}
		err := config.Validate(conf)
		require.Error(t, err)
	})
}

func newTestPoolConf() (InstancePoolConfig, *coremock.Gun) {
	gun := &coremock.Gun{}
	gun.On("Bind", mock.Anything, mock.Anything).Return(nil)
	gun.On("Shoot", mock.Anything)
	conf := InstancePoolConfig{
		Provider:   provider.NewNum(-1),
		Aggregator: aggregator.NewTest(),
		NewGun: func() (core.Gun, error) {
			return gun, nil
		},
		NewRPSSchedule: func() (core.Schedule, error) {
			return schedule.NewOnce(1), nil
		},
		StartupSchedule: schedule.NewOnce(1),
	}
	return conf, gun
}

func Test_InstancePool(t *testing.T) {
	var (
		gun    *coremock.Gun
		conf   InstancePoolConfig
		ctx    context.Context
		cancel context.CancelFunc

		waitDoneCalled atomic.Bool
		onWaitDone     func()

		p *instancePool
	)

	// Conf for starting only instance.
	var beforeEach = func() {
		conf, gun = newTestPoolConf()
		onWaitDone = func() {
			old := waitDoneCalled.Swap(true)
			if old {
				panic("double on wait done call")
			}
		}
		waitDoneCalled.Store(false)
		ctx, cancel = context.WithCancel(context.Background())
	}
	var justBeforeEach = func() {
		metrics := newTestMetrics()
		p = newPool(newNopLogger(), metrics, onWaitDone, conf)
	}
	_ = cancel

	t.Run("shoot ok", func(t *testing.T) {
		beforeEach()
		justBeforeEach()

		err := p.Run(ctx)
		require.NoError(t, err)
		gun.AssertExpectations(t)
		require.True(t, waitDoneCalled.Load())
	})

	t.Run("context canceled", func(t *testing.T) {
		var (
			blockShoot sync.WaitGroup
		)
		var beforeEachContext = func() {
			blockShoot.Add(1)
			prov := &coremock.Provider{}
			prov.On("Run", mock.Anything, mock.Anything).
				Return(func(startCtx context.Context, deps core.ProviderDeps) error {
					<-startCtx.Done()
					return nil
				})
			prov.On("Acquire").Return(func() (core.Ammo, bool) {
				cancel()
				blockShoot.Wait()
				return struct{}{}, true
			})
			conf.Provider = prov
		}

		beforeEach()
		beforeEachContext()
		justBeforeEach()

		err := p.Run(ctx)
		require.Equal(t, context.Canceled, err)
		gun.AssertNotCalled(t, "Shoot")
		assert.False(t, waitDoneCalled.Load())
		blockShoot.Done()

		tick := time.NewTicker(100 * time.Millisecond)
		i := 0
		for range tick.C {
			if waitDoneCalled.Load() {
				break
			}
			if i > 6 {
				break
			}
			i++
		}
		tick.Stop()
		assert.True(t, waitDoneCalled.Load()) //TODO: eventually
	})

	t.Run("provider failed", func(t *testing.T) {
		beforeEach()

		var (
			failErr           = errors.New("test err")
			blockShootAndAggr sync.WaitGroup
		)
		blockShootAndAggr.Add(1)
		prov := &coremock.Provider{}
		prov.On("Run", mock.Anything, mock.Anything).
			Return(func(context.Context, core.ProviderDeps) error {
				return failErr
			})
		prov.On("Acquire").Return(func() (core.Ammo, bool) {
			blockShootAndAggr.Wait()
			return nil, false
		})
		conf.Provider = prov
		aggr := &coremock.Aggregator{}
		aggr.On("Run", mock.Anything, mock.Anything).
			Return(func(context.Context, core.AggregatorDeps) error {
				blockShootAndAggr.Wait()
				return nil
			})
		conf.Aggregator = aggr

		justBeforeEach()

		err := p.Run(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, failErr.Error())
		gun.AssertNotCalled(t, "Shoot")

		assert.False(t, waitDoneCalled.Load())
		blockShootAndAggr.Done()

		tick := time.NewTicker(100 * time.Millisecond)
		i := 0
		for range tick.C {
			if waitDoneCalled.Load() {
				break
			}
			if i > 6 {
				break
			}
			i++
		}
		tick.Stop()
		assert.True(t, waitDoneCalled.Load()) //TODO: eventually
	})

	t.Run("aggregator failed", func(t *testing.T) {
		beforeEach()
		failErr := errors.New("test err")
		aggr := &coremock.Aggregator{}
		aggr.On("Run", mock.Anything, mock.Anything).Return(failErr)
		conf.Aggregator = aggr
		justBeforeEach()

		err := p.Run(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, failErr.Error())
		tick := time.NewTicker(100 * time.Millisecond)
		i := 0
		for range tick.C {
			if waitDoneCalled.Load() {
				break
			}
			if i > 6 {
				break
			}
			i++
		}
		tick.Stop()
		assert.True(t, waitDoneCalled.Load()) //TODO: eventually
	})

	t.Run("start instances failed", func(t *testing.T) {
		failErr := errors.New("test err")
		beforeEach()
		conf.NewGun = func() (core.Gun, error) {
			return nil, failErr
		}
		justBeforeEach()

		err := p.Run(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, failErr.Error())
		tick := time.NewTicker(100 * time.Millisecond)
		i := 0
		for range tick.C {
			if waitDoneCalled.Load() {
				break
			}
			if i > 6 {
				break
			}
			i++
		}
		tick.Stop()
		assert.True(t, waitDoneCalled.Load()) //TODO: eventually
	})
}

func Test_MultipleInstance(t *testing.T) {
	t.Run("out of ammo - instance start is canceled", func(t *testing.T) {
		conf, _ := newTestPoolConf()
		conf.Provider = provider.NewNum(3)
		conf.NewRPSSchedule = func() (core.Schedule, error) {
			return schedule.NewUnlimited(time.Hour), nil
		}
		conf.StartupSchedule = schedule.NewComposite(
			schedule.NewOnce(2),
			schedule.NewConst(1, 5*time.Second),
		)
		pool := newPool(newNopLogger(), newTestMetrics(), nil, conf)
		ctx := context.Background()

		err := pool.Run(ctx)
		require.NoError(t, err)
		require.True(t, pool.metrics.InstanceStart.Get() == 3)
	})

	t.Run("when provider run done it does not mean out of ammo; instance start is not canceled", func(t *testing.T) {
		conf, _ := newTestPoolConf()
		conf.Provider = provider.NewNumBuffered(3)
		conf.NewRPSSchedule = func() (core.Schedule, error) {
			return schedule.NewOnce(1), nil
		}
		conf.StartupSchedule = schedule.NewOnce(3)
		pool := newPool(newNopLogger(), newTestMetrics(), nil, conf)
		ctx := context.Background()

		err := pool.Run(ctx)
		require.NoError(t, err)
		require.True(t, pool.metrics.InstanceStart.Get() <= 3)
	})

	t.Run("out of RPS - instance start is canceled", func(t *testing.T) {
		conf, _ := newTestPoolConf()
		conf.NewRPSSchedule = func() (core.Schedule, error) {
			return schedule.NewOnce(5), nil
		}
		conf.StartupSchedule = schedule.NewComposite(
			schedule.NewOnce(2),
			schedule.NewConst(1, 2*time.Second),
		)
		pool := newPool(newNopLogger(), newTestMetrics(), nil, conf)
		ctx := context.Background()

		err := pool.Run(ctx)
		require.NoError(t, err)
		require.True(t, pool.metrics.InstanceStart.Get() <= 3)
	})
}

// TODO instance start canceled after out of ammo
// TODO instance start cancdled after RPS finish

func Test_Engine(t *testing.T) {
	var (
		gun1, gun2 *coremock.Gun
		confs      []InstancePoolConfig
		ctx        context.Context
		cancel     context.CancelFunc
		engine     *Engine
	)
	_ = cancel
	var beforeEach = func() {
		confs = make([]InstancePoolConfig, 2)
		confs[0], gun1 = newTestPoolConf()
		confs[1], gun2 = newTestPoolConf()
		ctx, cancel = context.WithCancel(context.Background())
	}

	var justBeforeEach = func() {
		metrics := newTestMetrics()
		engine = New(newNopLogger(), metrics, Config{confs})
	}

	t.Run("shoot ok", func(t *testing.T) {
		beforeEach()
		justBeforeEach()

		err := engine.Run(ctx)
		require.NoError(t, err)
		gun1.AssertExpectations(t)
		gun2.AssertExpectations(t)
	})

	t.Run("context canceled", func(t *testing.T) {

		// Cancel context on ammo acquire, an check that engine returns before
		// instance finish.
		var (
			blockPools sync.WaitGroup
		)
		var beforeEachCtx = func() {
			blockPools.Add(1)
			for i := range confs {
				prov := &coremock.Provider{}
				prov.On("Run", mock.Anything, mock.Anything).
					Return(func(startCtx context.Context, deps core.ProviderDeps) error {
						<-startCtx.Done()
						blockPools.Wait()
						return nil
					})
				prov.On("Acquire").Return(func() (core.Ammo, bool) {
					cancel()
					blockPools.Wait()
					return struct{}{}, true
				})
				confs[i].Provider = prov
			}
		}
		beforeEach()
		beforeEachCtx()
		justBeforeEach()

		err := engine.Run(ctx)
		require.Equal(t, err, context.Canceled)
		awaited := make(chan struct{})
		go func() {
			defer close(awaited)
			engine.Wait()
		}()

		assert.False(t, IsClosed(awaited))
		blockPools.Done()

		tick := time.NewTicker(100 * time.Millisecond)
		i := 0
		for range tick.C {
			if IsClosed(awaited) {
				break
			}
			if i > 6 {
				break
			}
			i++
		}
		tick.Stop()
		assert.True(t, IsClosed(awaited)) //TODO: eventually
	})

	t.Run("one pool failed", func(t *testing.T) {
		beforeEach()
		var (
			failErr = errors.New("test err")
		)
		aggr := &coremock.Aggregator{}
		aggr.On("Run", mock.Anything, mock.Anything).Return(failErr)
		confs[0].Aggregator = aggr

		justBeforeEach()

		err := engine.Run(ctx)
		require.Error(t, err)
		require.ErrorContains(t, err, failErr.Error())
		engine.Wait()
	})
}

func Test_BuildInstanceSchedule(t *testing.T) {
	t.Run("per instance schedule", func(t *testing.T) {
		conf, _ := newTestPoolConf()
		conf.RPSPerInstance = true
		pool := newPool(newNopLogger(), newTestMetrics(), nil, conf)
		newInstanceSchedule, err := pool.buildNewInstanceSchedule(context.Background(), func() {
			panic("should not be called")
		})
		require.NoError(t, err)

		val1 := reflect.ValueOf(newInstanceSchedule)
		val2 := reflect.ValueOf(conf.NewRPSSchedule)
		require.Equal(t, val1.Pointer(), val2.Pointer())
	})

	t.Run("shared schedule create failed", func(t *testing.T) {
		conf, _ := newTestPoolConf()
		scheduleCreateErr := errors.New("test err")
		conf.NewRPSSchedule = func() (core.Schedule, error) {
			return nil, scheduleCreateErr
		}
		pool := newPool(newNopLogger(), newTestMetrics(), nil, conf)
		newInstanceSchedule, err := pool.buildNewInstanceSchedule(context.Background(), func() {
			panic("should not be called")
		})

		require.Error(t, err)
		require.Equal(t, err, scheduleCreateErr)
		require.Nil(t, newInstanceSchedule)
	})

	t.Run("shared schedule work", func(t *testing.T) {
		conf, _ := newTestPoolConf()
		var newScheduleCalled bool
		conf.NewRPSSchedule = func() (core.Schedule, error) {
			require.False(t, newScheduleCalled)
			newScheduleCalled = true
			return schedule.NewOnce(1), nil
		}
		pool := newPool(newNopLogger(), newTestMetrics(), nil, conf)
		ctx, cancel := context.WithCancel(context.Background())
		newInstanceSchedule, err := pool.buildNewInstanceSchedule(context.Background(), cancel)
		require.NoError(t, err)

		schedule, err := newInstanceSchedule()
		require.NoError(t, err)

		assert.False(t, IsClosed(ctx.Done()))
		_, ok := schedule.Next()
		assert.True(t, ok)
		assert.False(t, IsClosed(ctx.Done()))
		_, ok = schedule.Next()
		assert.False(t, ok)
		assert.True(t, IsClosed(ctx.Done()))
	})
}

func IsClosed(actual any) (success bool) {
	if !isChan(actual) {
		return false
	}
	channelValue := reflect.ValueOf(actual)
	channelType := reflect.TypeOf(actual)
	if channelType.ChanDir() == reflect.SendDir {
		return false
	}

	winnerIndex, _, open := reflect.Select([]reflect.SelectCase{
		{Dir: reflect.SelectRecv, Chan: channelValue},
		{Dir: reflect.SelectDefault},
	})

	var closed bool
	if winnerIndex == 0 {
		closed = !open
	} else if winnerIndex == 1 {
		closed = false
	}

	return closed
}

func isChan(a interface{}) bool {
	if isNil(a) {
		return false
	}
	return reflect.TypeOf(a).Kind() == reflect.Chan
}

func isNil(a interface{}) bool {
	if a == nil {
		return true
	}

	switch reflect.TypeOf(a).Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return reflect.ValueOf(a).IsNil()
	}

	return false
}

type TestLogWriter struct {
	t *testing.T
}

func (w *TestLogWriter) Write(p []byte) (n int, err error) {
	w.t.Helper()
	w.t.Log(string(p))
	return len(p), nil
}

func newTestLogger(t *testing.T) *zap.Logger {
	conf := zap.NewDevelopmentConfig()
	enc := zapcore.NewConsoleEncoder(conf.EncoderConfig)
	core := zapcore.NewCore(enc, zapcore.AddSync(&TestLogWriter{t: t}), zap.DebugLevel)
	log := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zap.DPanicLevel))
	return log
}

func newNopLogger() *zap.Logger {
	core := zapcore.NewNopCore()
	log := zap.New(core)
	return log
}

func newTestMetrics() Metrics {
	return Metrics{
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
	}
}
