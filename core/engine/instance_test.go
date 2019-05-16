package engine

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/yandex/pandora/core"
	coremock "github.com/yandex/pandora/core/mocks"
	"github.com/yandex/pandora/core/schedule"
	"github.com/yandex/pandora/lib/ginkgoutil"
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

		ins          *instance
		insCreateErr error

		newSchedule func() (core.Schedule, error)
		newGun      func() (core.Gun, error)
	)

	BeforeEach(func() {
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
	})

	JustBeforeEach(func() {
		deps := instanceDeps{
			aggregator,
			newSchedule,
			newGun,
			instanceSharedDeps{
				provider,
				metrics,
			},
		}
		ins, insCreateErr = newInstance(ctx, ginkgoutil.NewLogger(), 0, deps)
	})

	AfterEach(func() {
		if newGunErr == nil && newScheduleErr == nil {
			Expect(metrics.InstanceStart.Get()).To(BeEquivalentTo(1))
			Expect(metrics.InstanceFinish.Get()).To(BeEquivalentTo(1))
		}
	})

	Context("all ok", func() {
		BeforeEach(func() {
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
		})
		JustBeforeEach(func() {
			Expect(insCreateErr).NotTo(HaveOccurred())
		})
		It("start ok", func() {
			err := ins.Run(ctx)
			Expect(err).NotTo(HaveOccurred())
			ginkgoutil.AssertExpectations(gun, provider)
		}, 2)

		Context("gun implements io.Closer", func() {
			var closeGun mockGunCloser
			BeforeEach(func() {
				closeGun = mockGunCloser{gun}
				closeGun.On("Close").Return(nil)
				newGun = func() (core.Gun, error) {
					return closeGun, nil
				}
			})
			It("close called on instance close", func() {
				err := ins.Run(ctx)
				Expect(err).NotTo(HaveOccurred())
				ginkgoutil.AssertNotCalled(closeGun, "Close")
				err = ins.Close()
				Expect(err).NotTo(HaveOccurred())
				ginkgoutil.AssertExpectations(closeGun, provider)
			})

		})
	})

	Context("context canceled after run", func() {
		BeforeEach(func() {
			ctx, _ = context.WithTimeout(ctx, 10*time.Millisecond)
			sched := sched.(*coremock.Schedule)
			sched.On("Next").Return(time.Now().Add(5*time.Second), true)
			sched.On("Left").Return(1)
			gun.On("Bind", aggregator, mock.Anything).Return(nil)
			provider.On("Acquire").Return(struct{}{}, true)
		})
		It("start fail", func() {
			err := ins.Run(ctx)
			Expect(err).To(Equal(context.DeadlineExceeded))
			ginkgoutil.AssertExpectations(gun, provider)
		}, 2)

	})

	Context("context canceled before run", func() {
		BeforeEach(func() {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			cancel()
			gun.On("Bind", aggregator, mock.Anything).Return(nil)
		})
		It("nothing acquired and schedule not started", func() {
			err := ins.Run(ctx)
			Expect(err).To(Equal(context.Canceled))
			ginkgoutil.AssertExpectations(gun, provider)
		}, 2)

	})

	Context("schedule create failed", func() {
		BeforeEach(func() {
			sched = nil
			newScheduleErr = errors.New("test err")
		})
		It("instance create failed", func() {
			Expect(insCreateErr).To(Equal(newScheduleErr))
		})
	})

	Context("gun create failed", func() {
		BeforeEach(func() {
			gun = nil
			newGunErr = errors.New("test err")
		})
		It("instance create failed", func() {
			Expect(insCreateErr).To(Equal(newGunErr))
		})
	})

})

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
