package engine

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/mocks"
	"github.com/yandex/pandora/core/schedule"
	"github.com/yandex/pandora/lib/testutil"
)

var _ = Describe("Instance", func() {
	var (
		provider       *coremock.Provider
		aggregator     *coremock.Aggregator
		sched          core.Schedule
		newScheduleErr error
		gun            *coremock.Gun
		newGunErr      error
		ctx            context.Context
		metrics        Metrics

		deps instanceDeps
		ins  *instance
	)
	const instanceId = "id"

	BeforeEach(func() {
		provider = &coremock.Provider{}
		aggregator = &coremock.Aggregator{}
		gun = &coremock.Gun{}
		newGunErr = nil
		sched = &coremock.Schedule{}
		newScheduleErr = nil
		ctx = context.Background()
	})

	JustBeforeEach(func() {
		metrics = newTestMetrics()
		deps = instanceDeps{
			provider,
			aggregator,
			func() (core.Schedule, error) { return sched, newScheduleErr },
			func() (core.Gun, error) { return gun, newGunErr },
			metrics,
		}
		ins = newInstance(testutil.NewLogger(), instanceId, deps)
	})

	AfterEach(func() {
		Expect(metrics.InstanceStart.Get()).To(BeEquivalentTo(1))
		Expect(metrics.InstanceFinish.Get()).To(BeEquivalentTo(1))
	})

	Context("all ok", func() {
		BeforeEach(func() {
			const times = 5
			sched = schedule.NewOnce(times)
			gun.On("Bind", aggregator).Once()
			var acquired int
			provider.On("Acquire").Return(func() (core.Ammo, bool) {
				acquired++
				return acquired, true
			}) // TODO(skipor): .Times(times) after fix one extra ammo consume at schedule end.
			for i := 1; i <= times; i++ {
				gun.On("Shoot", ctx, i).Once()
				provider.On("Release", i).Once()
			}
		})
		It("start ok", func() {
			err := ins.Run(ctx)
			Expect(err).NotTo(HaveOccurred())
			testutil.AssertExpectations(gun, provider)
		}, 2)

		It("gun implements io.Closer", func() {
			closeGun := &mockGunCloser{*gun}
			closeGun.On("Close").Return(nil)
			deps.newGun = func() (core.Gun, error) {
				return closeGun, nil
			}
			ins = newInstance(testutil.NewLogger(), instanceId, deps)
			err := ins.Run(ctx)
			Expect(err).NotTo(HaveOccurred())
			testutil.AssertExpectations(closeGun, provider)
		})

	})

	Context("context canceled", func() {
		BeforeEach(func() {
			ctx, _ = context.WithTimeout(ctx, 5*time.Millisecond)
			sched.(*coremock.Schedule).On("Next").
				Return(func() (time.Time, bool) {
					return time.Now().Add(5 * time.Second), true
				})
			gun.On("Bind", aggregator)
			provider.On("Acquire").Return(struct{}{}, true)
		})
		It("start fail", func() {
			err := ins.Run(ctx)
			Expect(err).To(Equal(context.DeadlineExceeded))
			testutil.AssertExpectations(gun, provider)
		}, 2)
	})

	Context("schedule create failed", func() {
		BeforeEach(func() {
			sched = nil
			newScheduleErr = errors.New("test err")
		})
		It("start fail", func() {
			err := ins.Run(ctx)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(newScheduleErr.Error()))
		})
	})

	Context("gun create failed", func() {
		BeforeEach(func() {
			gun = nil
			newGunErr = errors.New("test err")
		})
		It("start fail", func() {
			err := ins.Run(ctx)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(newGunErr.Error()))
		})
	})

})

type mockGunCloser struct {
	coremock.Gun
}

func (_m *mockGunCloser) Close() error {
	ret := _m.Called()

	var r0 error
	if rf, ok := ret.Get(0).(func() error); ok {
		r0 = rf()
	} else {
		r0 = ret.Error(0)
	}
	return r0
}
