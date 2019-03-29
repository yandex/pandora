package engine

import (
	"context"
	"sync"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"go.uber.org/atomic"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator"
	"github.com/yandex/pandora/core/config"
	coremock "github.com/yandex/pandora/core/mocks"
	"github.com/yandex/pandora/core/provider"
	"github.com/yandex/pandora/core/schedule"
	"github.com/yandex/pandora/lib/ginkgoutil"
)

var _ = Describe("config validation", func() {
	It("dive validation", func() {
		conf := Config{
			Pools: []InstancePoolConfig{
				{},
			},
		}
		err := config.Validate(conf)
		Expect(err).To(HaveOccurred())
	})

	It("pools required", func() {
		conf := Config{}
		err := config.Validate(conf)
		Expect(err).To(HaveOccurred())
	})

})

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

var _ = Describe("instance pool", func() {
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
	BeforeEach(func() {
		conf, gun = newTestPoolConf()
		onWaitDone = func() {
			old := waitDoneCalled.Swap(true)
			if old {
				panic("double on wait done call")
			}
		}
		waitDoneCalled.Store(false)
		ctx, cancel = context.WithCancel(context.Background())
	})

	JustBeforeEach(func() {
		metrics := newTestMetrics()
		p = newPool(ginkgoutil.NewLogger(), metrics, onWaitDone, conf)
	})

	Context("shoot ok", func() {
		It("", func() {
			err := p.Run(ctx)
			Expect(err).To(BeNil())
			ginkgoutil.AssertExpectations(gun)
			Expect(waitDoneCalled.Load()).To(BeTrue())
		}, 1)
	})

	Context("context canceled", func() {
		var (
			blockShoot sync.WaitGroup
		)
		BeforeEach(func() {
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
		})
		It("", func() {
			err := p.Run(ctx)
			Expect(err).To(Equal(context.Canceled))
			ginkgoutil.AssertNotCalled(gun, "Shoot")
			Expect(waitDoneCalled.Load()).To(BeFalse())
			blockShoot.Done()
			Eventually(waitDoneCalled.Load).Should(BeTrue())
		}, 1)
	})

	Context("provider failed", func() {
		var (
			failErr           = errors.New("test err")
			blockShootAndAggr sync.WaitGroup
		)
		BeforeEach(func() {
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
		})
		It("", func() {
			err := p.Run(ctx)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(failErr.Error()))
			ginkgoutil.AssertNotCalled(gun, "Shoot")
			Consistently(waitDoneCalled.Load, 0.1).Should(BeFalse())
			blockShootAndAggr.Done()
			Eventually(waitDoneCalled.Load).Should(BeTrue())
		})
	})

	Context("aggregator failed", func() {
		failErr := errors.New("test err")
		BeforeEach(func() {
			aggr := &coremock.Aggregator{}
			aggr.On("Run", mock.Anything, mock.Anything).Return(failErr)
			conf.Aggregator = aggr
		})
		It("", func() {
			err := p.Run(ctx)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(failErr.Error()))
			Eventually(waitDoneCalled.Load).Should(BeTrue())
		}, 1)
	})

	Context("start instances failed", func() {
		failErr := errors.New("test err")
		BeforeEach(func() {
			conf.NewGun = func() (core.Gun, error) {
				return nil, failErr
			}
		})
		It("", func() {
			err := p.Run(ctx)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(failErr.Error()))
			Eventually(waitDoneCalled.Load).Should(BeTrue())
		}, 1)
	})

})

var _ = Describe("multiple instance", func() {
	It("out of ammo - instance start is canceled", func() {
		conf, _ := newTestPoolConf()
		conf.Provider = provider.NewNum(3)
		conf.NewRPSSchedule = func() (core.Schedule, error) {
			return schedule.NewUnlimited(time.Hour), nil
		}
		conf.StartupSchedule = schedule.NewComposite(
			schedule.NewOnce(2),
			schedule.NewConst(1, 5*time.Second),
		)
		pool := newPool(ginkgoutil.NewLogger(), newTestMetrics(), nil, conf)
		ctx := context.Background()

		err := pool.Run(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(pool.metrics.InstanceStart.Get()).To(BeNumerically("<=", 3))
	}, 1)

	It("out of RPS - instance start is canceled", func() {
		conf, _ := newTestPoolConf()
		conf.NewRPSSchedule = func() (core.Schedule, error) {
			return schedule.NewOnce(5), nil
		}
		conf.StartupSchedule = schedule.NewComposite(
			schedule.NewOnce(2),
			schedule.NewConst(1, 2*time.Second),
		)
		pool := newPool(ginkgoutil.NewLogger(), newTestMetrics(), nil, conf)
		ctx := context.Background()

		err := pool.Run(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(pool.metrics.InstanceStart.Get()).To(BeNumerically("<=", 3))
	})

})

// TODO instance start canceled after out of ammo
// TODO instance start cancdled after RPS finish

var _ = Describe("engine", func() {
	var (
		gun1, gun2 *coremock.Gun
		confs      []InstancePoolConfig
		ctx        context.Context
		cancel     context.CancelFunc
		engine     *Engine
	)
	BeforeEach(func() {
		confs = make([]InstancePoolConfig, 2)
		confs[0], gun1 = newTestPoolConf()
		confs[1], gun2 = newTestPoolConf()
		ctx, cancel = context.WithCancel(context.Background())
	})

	JustBeforeEach(func() {
		metrics := newTestMetrics()
		engine = New(ginkgoutil.NewLogger(), metrics, Config{confs})
	})

	Context("shoot ok", func() {
		It("", func() {
			err := engine.Run(ctx)
			Expect(err).To(BeNil())
			ginkgoutil.AssertExpectations(gun1, gun2)
		})
	})

	Context("context canceled", func() {
		// Cancel context on ammo acquire, an check that engine returns before
		// instance finish.
		var (
			blockPools sync.WaitGroup
		)
		BeforeEach(func() {
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
		})

		It("", func() {
			err := engine.Run(ctx)
			Expect(err).To(Equal(context.Canceled))
			awaited := make(chan struct{})
			go func() {
				defer close(awaited)
				engine.Wait()
			}()
			Consistently(awaited, 0.1).ShouldNot(BeClosed())
			blockPools.Done()
			Eventually(awaited).Should(BeClosed())
		})
	})

	Context("one pool failed", func() {
		var (
			failErr = errors.New("test err")
		)
		BeforeEach(func() {
			aggr := &coremock.Aggregator{}
			aggr.On("Run", mock.Anything, mock.Anything).Return(failErr)
			confs[0].Aggregator = aggr
		})

		It("", func() {
			err := engine.Run(ctx)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(failErr.Error()))
			engine.Wait()
		}, 1)
	})
})

var _ = Describe("build instance schedule", func() {
	It("per instance schedule ", func() {
		conf, _ := newTestPoolConf()
		conf.RPSPerInstance = true
		pool := newPool(ginkgoutil.NewLogger(), newTestMetrics(), nil, conf)
		newInstanceSchedule, err := pool.buildNewInstanceSchedule(context.Background(), func() {
			Fail("should not be called")
		})
		Expect(err).NotTo(HaveOccurred())
		ginkgoutil.ExpectFuncsEqual(newInstanceSchedule, conf.NewRPSSchedule)
	})

	It("shared schedule create failed", func() {
		conf, _ := newTestPoolConf()
		scheduleCreateErr := errors.New("test err")
		conf.NewRPSSchedule = func() (core.Schedule, error) {
			return nil, scheduleCreateErr
		}
		pool := newPool(ginkgoutil.NewLogger(), newTestMetrics(), nil, conf)
		newInstanceSchedule, err := pool.buildNewInstanceSchedule(context.Background(), func() {
			Fail("should not be called")
		})
		Expect(err).To(Equal(scheduleCreateErr))
		Expect(newInstanceSchedule).To(BeNil())
	})

	It("shared schedule work", func() {
		conf, _ := newTestPoolConf()
		var newScheduleCalled bool
		conf.NewRPSSchedule = func() (core.Schedule, error) {
			Expect(newScheduleCalled).To(BeFalse())
			newScheduleCalled = true
			return schedule.NewOnce(1), nil
		}
		pool := newPool(ginkgoutil.NewLogger(), newTestMetrics(), nil, conf)
		ctx, cancel := context.WithCancel(context.Background())
		newInstanceSchedule, err := pool.buildNewInstanceSchedule(context.Background(), cancel)
		Expect(err).NotTo(HaveOccurred())

		schedule, err := newInstanceSchedule()
		Expect(err).NotTo(HaveOccurred())

		Expect(newInstanceSchedule()).To(Equal(schedule))

		Expect(ctx.Done()).NotTo(BeClosed())
		_, ok := schedule.Next()
		Expect(ok).To(BeTrue())
		Expect(ctx.Done()).NotTo(BeClosed())
		_, ok = schedule.Next()
		Expect(ok).To(BeFalse())
		Expect(ctx.Done()).To(BeClosed())
	})

})
