package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core"
	coremock "github.com/yandex/pandora/core/mocks"
	"github.com/yandex/pandora/core/schedule"
)

func Test_Instance(t *testing.T) {
	var (
		provider       *coremock.Provider
		aggregator     *coremock.Aggregator
		sched          core.Schedule
		newScheduleErr error
		gun            *coremock.Gun
		newGunErr      error
		ctx            context.Context
		metrics        Metrics

		ins          *instance
		insCreateErr error

		newSchedule func() (core.Schedule, error)
		newGun      func() (core.Gun, error)
	)

	var beforeEach = func() {
		provider = &coremock.Provider{}
		aggregator = &coremock.Aggregator{}
		gun = &coremock.Gun{}
		newGunErr = nil
		sched = &coremock.Schedule{}
		newScheduleErr = nil
		ctx = context.Background()
		metrics = newTestMetrics()
		newSchedule = func() (core.Schedule, error) { return sched, newScheduleErr }
		newGun = func() (core.Gun, error) { return gun, newGunErr }
	}

	var justBeforeEach = func() {
		deps := instanceDeps{

			newSchedule,
			newGun,
			instanceSharedDeps{
				provider,
				metrics,
				nil,
				aggregator,
				false,
			},
		}
		ins, insCreateErr = newInstance(ctx, newNopLogger(), "pool_0", 0, deps)
	}

	var afterEach = func() {
		if newGunErr == nil && newScheduleErr == nil {
			assert.Equal(t, int64(1), metrics.InstanceStart.Get())
			assert.Equal(t, int64(1), metrics.InstanceFinish.Get())
		}
	}

	t.Run("all ok", func(t *testing.T) {
		var beforeEachCtx = func() {
			const times = 5
			sched = schedule.NewOnce(times)
			gun.On("Bind", aggregator, mock.Anything).Return(nil).Once()
			var acquired int
			provider.On("Acquire").Return(func() (core.Ammo, bool) {
				acquired++
				return acquired, true
			}).Times(times)
			for i := 1; i <= times; i++ {
				gun.On("Shoot", i).Once()
				provider.On("Release", i).Once()
			}
		}
		var justBeforeEachCtx = func() {
			require.NoError(t, insCreateErr)
		}
		t.Run("start ok", func(t *testing.T) {
			beforeEach()
			beforeEachCtx()
			justBeforeEachCtx()
			justBeforeEach()

			err := ins.Run(ctx)
			require.NoError(t, err)
			gun.AssertExpectations(t)
			provider.AssertExpectations(t)

			afterEach()
		})

		t.Run("gun implements io.Closer / close called on instance close", func(t *testing.T) {
			beforeEach()
			beforeEachCtx()
			closeGun := mockGunCloser{gun}
			closeGun.On("Close").Return(nil)
			newGun = func() (core.Gun, error) {
				return closeGun, nil
			}
			justBeforeEachCtx()
			justBeforeEach()

			err := ins.Run(ctx)
			require.NoError(t, err)
			closeGun.AssertNotCalled(t, "Close")
			err = ins.Close()
			require.NoError(t, err)
			closeGun.AssertExpectations(t)
			provider.AssertExpectations(t)

			afterEach()
		})
	})

	t.Run("context canceled after run / start fail", func(t *testing.T) {
		beforeEach()

		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 10*time.Millisecond)
		_ = cancel
		sched := sched.(*coremock.Schedule)
		sched.On("Next").Return(time.Now().Add(5*time.Second), true)
		sched.On("Left").Return(1)
		gun.On("Bind", aggregator, mock.Anything).Return(nil)
		provider.On("Acquire").Return(struct{}{}, true)
		provider.On("Release", mock.Anything).Return()

		justBeforeEach()

		err := ins.Run(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
		gun.AssertExpectations(t)
		provider.AssertExpectations(t)

		afterEach()
	})

	t.Run("context canceled before run / nothing acquired and schedule not started", func(t *testing.T) {
		beforeEach()
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		cancel()
		gun.On("Bind", aggregator, mock.Anything).Return(nil)
		justBeforeEach()

		err := ins.Run(ctx)
		require.Equal(t, context.Canceled, err)
		gun.AssertExpectations(t)
		provider.AssertExpectations(t)

		afterEach()
	})

	t.Run("schedule create failed / instance create failed", func(t *testing.T) {
		beforeEach()
		sched = nil
		newScheduleErr = errors.New("test err")
		justBeforeEach()

		require.Equal(t, newScheduleErr, insCreateErr)

		afterEach()
	})

	t.Run("gun create failed / instance create failed", func(t *testing.T) {
		beforeEach()
		gun = nil
		newGunErr = errors.New("test err")
		justBeforeEach()

		require.Equal(t, newGunErr, insCreateErr)
		afterEach()
	})
}

type mockGunCloser struct {
	*coremock.Gun
}

func (_m mockGunCloser) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}
	return r0
}
