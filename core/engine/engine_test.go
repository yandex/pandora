package engine

import (
	"context"
	"errors"
	"sync"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
	"go.uber.org/atomic"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregate"
	"github.com/yandex/pandora/core/mocks"
	"github.com/yandex/pandora/core/provider"
	"github.com/yandex/pandora/core/schedule"
	"github.com/yandex/pandora/lib/testutil"
)

func newTestPoolConf() (InstancePoolConfig, *coremock.Gun) {
	gun := &coremock.Gun{}
	gun.On("Bind", mock.Anything)
	gun.On("Shoot", mock.Anything, mock.Anything)
	conf := InstancePoolConfig{
		Provider:   provider.NewNum(-1),
		Aggregator: aggregate.NewDiscard(),
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
		p = newPool(testutil.NewLogger(), metrics, onWaitDone, conf)
	})

	Context("shoot ok", func() {
		It("", func() {
			err := p.Run(ctx)
			Expect(err).To(BeNil())
			testutil.AssertExpectations(gun)
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
			prov.On("Start", mock.Anything).
				Return(func(startCtx context.Context) error {
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
			testutil.AssertNotCalled(gun, "Shoot")
			Expect(waitDoneCalled.Load()).To(BeFalse())
			blockShoot.Done()
			Eventually(waitDoneCalled.Load).Should(BeTrue())
		}, 1)
	})

	Context("provider failed", func() {
		failErr := errors.New("test err")
		BeforeEach(func() {
			startCtxC := make(chan context.Context, 1)
			prov := &coremock.Provider{}
			prov.On("Start", mock.Anything).
				Return(func(startCtx context.Context) error {
				startCtxC <- startCtx
				return failErr
			})
			prov.On("Acquire").Return(func() (core.Ammo, bool) {
				startCtx := <-startCtxC
				<-startCtx.Done()
				return nil, false
			})
			conf.Provider = prov
		})
		It("", func() {
			err := p.Run(ctx)
			Expect(err).ToNot(BeNil())
			Expect(err.Error()).To(ContainSubstring(failErr.Error()))
			testutil.AssertNotCalled(gun, "Shoot")
			Eventually(waitDoneCalled.Load).Should(BeTrue())
		}, 1)
	})

	Context("aggregator failed", func() {
		failErr := errors.New("test err")
		BeforeEach(func() {
			aggr := &coremock.Aggregator{}
			aggr.On("Start", mock.Anything).Return(failErr)
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

var _ = FDescribe("engine", func() {
	var (
		gun1, gun2 *coremock.Gun
		//conf1, conf2 InstancePoolConfig
		confs  []InstancePoolConfig
		ctx    context.Context
		cancel context.CancelFunc
		engine *Engine
	)
	BeforeEach(func() {
		confs = make([]InstancePoolConfig, 2)
		confs[0], gun1 = newTestPoolConf()
		confs[1], gun2 = newTestPoolConf()
		ctx, cancel = context.WithCancel(context.Background())
	})

	JustBeforeEach(func() {
		metrics := newTestMetrics()
		engine = New(testutil.NewLogger(), metrics, Config{confs})
	})

	Context("shoot ok", func() {
		It("", func() {
			err := engine.Run(ctx)
			Expect(err).To(BeNil())
			testutil.AssertExpectations(gun1, gun2)
		})
	})

	Context("context canceled", func() {
		// Cancel context on ammo acquire, an check that engine returns before
		// instance finish.
		var (
			blockShoot sync.WaitGroup
		)
		BeforeEach(func() {
			blockShoot.Add(1)
			for i := range confs {
				prov := &coremock.Provider{}
				prov.On("Start", mock.Anything).
					Return(func(startCtx context.Context) error {
					<-startCtx.Done()
					return nil
				})
				prov.On("Acquire").Return(func() (core.Ammo, bool) {
					cancel()
					blockShoot.Wait()
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
			blockShoot.Done()
			Eventually(awaited).Should(BeClosed())
		})
	})

	Context("one pool failed", func() {
		var (
			failErr = errors.New("test err")
			//blockShoot sync.WaitGroup
		)
		BeforeEach(func() {
			aggr := &coremock.Aggregator{}
			aggr.On("Start", mock.Anything).Return(failErr)
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
