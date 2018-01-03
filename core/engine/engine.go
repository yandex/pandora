// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package engine

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"go.uber.org/zap"

	"github.com/pkg/errors"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coreutil"
	"github.com/yandex/pandora/lib/monitoring"
)

type Config struct {
	Pools []InstancePoolConfig `config:"pools"`
}

type InstancePoolConfig struct {
	Id              string
	Provider        core.Provider                 `config:"ammo"`
	Aggregator      core.Aggregator               `config:"result"`
	NewGun          func() (core.Gun, error)      `config:"gun"`
	RPSPerInstance  bool                          `config:"rps-per-instance"`
	NewRPSSchedule  func() (core.Schedule, error) `config:"rps"`
	StartupSchedule core.Schedule                 `config:"startup"`
}

// TODO(skipor): use something github.com/rcrowley/go-metrics based.
// Its high level primitives like Meter can be not fast enough, but EWMAs
// and Counters should good for that.
type Metrics struct {
	Request        *monitoring.Counter
	Response       *monitoring.Counter
	InstanceStart  *monitoring.Counter
	InstanceFinish *monitoring.Counter
}

func New(log *zap.Logger, m Metrics, conf Config) *Engine {
	return &Engine{log: log, config: conf, metrics: m}
}

type Engine struct {
	log     *zap.Logger
	config  Config
	metrics Metrics
	wait    sync.WaitGroup
}

// Run runs all instance pools. Run blocks until fail happen, or all pools
// subroutines are successfully finished.
func (e *Engine) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer func() {
		e.log.Info("Engine finished")
		cancel()
	}()

	runRes := make(chan runResult, 1)
	for i, conf := range e.config.Pools {
		if conf.Id == "" {
			conf.Id = fmt.Sprintf("pool_%v", i)
		}
		e.wait.Add(1)
		pool := newPool(e.log, e.metrics, e.wait.Done, conf)
		go func() {
			err := pool.Run(ctx)
			select {
			case runRes <- runResult{pool.Id, err}:
			case <-ctx.Done():
				pool.log.Info("Pool run result suppressed",
					zap.String("id", pool.Id), zap.Error(err))
			}
		}()
	}

	for i := 0; i < len(e.config.Pools); i++ {
		select {
		case res := <-runRes:
			e.log.Debug("Pool awaited", zap.Int("awaited", i), zap.String("id", res.Id), zap.Error(res.Err))
			if res.Err != nil {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}
				return errors.WithMessage(res.Err, fmt.Sprintf("%q pool run failed", res.Id))
			}
		case <-ctx.Done():
			e.log.Info("Engine run canceled")
			return ctx.Err()
		}
	}
	return nil
}

// Wait blocks until all run engine tasks are finished.
// Useful only in case of fail, because successful run awaits all started tasks.
func (e *Engine) Wait() {
	e.wait.Wait()
}

func newPool(log *zap.Logger, m Metrics, onWaitDone func(), conf InstancePoolConfig) *instancePool {
	log = log.With(zap.String("pool", conf.Id))
	return &instancePool{log, m, onWaitDone, conf}
}

type instancePool struct {
	log        *zap.Logger
	metrics    Metrics
	onWaitDone func()
	InstancePoolConfig
}

// Run start instance pool. Run blocks until fail happen, or all instances finish.
// What's going on:
// Provider and Aggregator are started in separate goroutines.
// Instances create due to schedule is started in separate goroutine.
// Every new instance started in separate goroutine.
// When all instances are finished, Aggregator and Provider contexts are canceled,
// and their execution results are awaited.
// If error happen or Run context has been canceled, Run returns non-nil error immediately,
// remaining results awaiting goroutine in background, that will call onWaitDone callback,
// when all started subroutines will be finished.
func (p *instancePool) Run(ctx context.Context) error {
	p.log.Info("Pool started")
	originalCtx := ctx                               // Canceled only in case of other pool fail.
	ctx, cancel := context.WithCancel(ctx)           // Canceled in case of fail, or all instances finish.
	startCtx, startCancel := context.WithCancel(ctx) // Canceled also on out of ammo, and finish of shared RPS schedule.
	// TODO(skipor): err should not be visible bellow. Too complex. Refactor.
	newInstanceSchedule, err := p.buildNewInstanceSchedule(startCtx, startCancel)
	if err != nil {
		return err
	}
	defer func() {
		p.log.Info("Pool run finished")
		cancel()
	}()

	var (
		providerErr   = make(chan error, 1)
		aggregatorErr = make(chan error, 1)
		runRes        = make(chan runResult, 1)
		startRes      = make(chan startResult, 1)
	)
	go func() {
		providerErr <- p.Provider.Run(ctx)
	}()
	go func() {
		aggregatorErr <- p.Aggregator.Run(ctx)
	}()
	go func() {
		// Running in separate goroutine, so we can cancel instance creation.
		started, err := p.startInstances(startCtx, ctx, newInstanceSchedule, runRes)
		startRes <- startResult{started, err}
	}()

	awaitErr := make(chan error)
	errAwaited := func(err error) {
		select {
		case awaitErr <- err:
		case <-ctx.Done():
			if err != ctx.Err() {
				p.log.Debug("Error suppressed after run cancel", zap.Error(err))
			}
		}
	}
	// Await all launched in separate goroutine, so we can return first execution
	// error, and still wait results in background.
	go func() {
		defer func() {
			p.log.Debug("Pool wait finished")
			close(awaitErr)
			if p.onWaitDone != nil {
				p.onWaitDone()
			}
		}()
		const subroutines = 4 // Provider, Aggregator, instance start, instance run.
		var (
			toWait           = subroutines
			startedInstances = -1 // Undefined, until instance start finish.
			awaitedInstances = 0
		)
		checkAllInstancesAreFinished := func() {
			startFinished := startRes == nil
			if !startFinished || awaitedInstances < startedInstances {
				return
			}
			close(runRes) // Nothing should be sent more.
			runRes = nil
			toWait--
			p.log.Info("All instances runs awaited.", zap.Int("awaited", awaitedInstances))
			cancel() // Signal to provider and aggregator, that pool run is finished.
		}

		for toWait > 0 {
			select {
			case err := <-providerErr:
				providerErr = nil
				// TODO(skipor): not wait for provider, to return success result?
				toWait--
				p.log.Debug("Provider awaited", zap.Error(err))
				if nonCtxErr(ctx, err) {
					errAwaited(errors.WithMessage(err, "provider failed"))
				}
				if err == nil && startRes != nil {
					p.log.Debug("Canceling instance start because out of ammo")
					startCancel()
				}
			case err := <-aggregatorErr:
				aggregatorErr = nil
				toWait--
				p.log.Debug("Aggregator awaited", zap.Error(err))
				if nonCtxErr(ctx, err) {
					errAwaited(errors.WithMessage(err, "aggregator failed"))
				}
			case res := <-startRes:
				startRes = nil
				toWait--
				startedInstances = res.Started
				p.log.Debug("Instances start awaited", zap.Int("started", startedInstances), zap.Error(res.Err))
				if nonCtxErr(startCtx, res.Err) {
					errAwaited(errors.WithMessage(res.Err, "instances start failed"))
				}
				checkAllInstancesAreFinished()
			case res := <-runRes:
				awaitedInstances++
				if ent := p.log.Check(zap.DebugLevel, "Instance run awaited"); ent != nil {
					ent.Write(zap.String("id", res.Id), zap.Error(res.Err), zap.Int("awaited", awaitedInstances))
				}
				if nonCtxErr(ctx, res.Err) {
					errAwaited(errors.WithMessage(res.Err, fmt.Sprintf("instance %q run failed", res.Id)))
				}
				checkAllInstancesAreFinished()
			}
		}
	}()

	select {
	case <-originalCtx.Done():
		p.log.Info("Pool execution canceled")
		return ctx.Err()
	case err, ok := <-awaitErr:
		if ok {
			p.log.Info("Pool failed. Canceling started tasks", zap.Error(err))
			return err
		}
		p.log.Info("Pool run finished successfully")
		return nil
	}
}

func (p *instancePool) startInstances(
	startCtx, runCtx context.Context,
	newInstanceSchedule func() (core.Schedule, error),
	runRes chan<- runResult) (started int, err error) {
	deps := instanceDeps{
		p.Aggregator,
		newInstanceSchedule,
		p.NewGun,
		instanceSharedDeps{p.Provider, p.metrics},
	}

	waiter := coreutil.NewWaiter(p.StartupSchedule, startCtx)

	// If create all instances asynchronously, and creation will fail, too many errors appears in log.
	ok := waiter.Wait()
	if !ok {
		err = startCtx.Err()
		return
	}
	firstInstance, err := newInstance(p.log, instanceId(0), deps)
	if err != nil {
		return
	}
	started++
	go func() {
		defer firstInstance.Close()
		runRes <- runResult{firstInstance.id, firstInstance.Run(runCtx)}
	}()

	for ; waiter.Wait(); started++ {
		id := strconv.Itoa(started)
		go func() {
			runRes <- runResult{id, runNewInstance(runCtx, p.log, id, deps)}
		}()
	}
	err = startCtx.Err()
	return
}

func (p *instancePool) buildNewInstanceSchedule(startCtx context.Context, cancelStart context.CancelFunc) (
	newInstanceSchedule func() (core.Schedule, error), err error,
) {
	if p.RPSPerInstance {
		newInstanceSchedule = p.NewRPSSchedule
		return
	}
	var sharedRPSSchedule core.Schedule
	sharedRPSSchedule, err = p.NewRPSSchedule()
	if err != nil {
		return
	}
	sharedRPSSchedule = coreutil.NewCallbackOnFinishSchedule(sharedRPSSchedule, func() {
		select {
		case <-startCtx.Done():
			p.log.Debug("RPS schedule has been finished")
			return
		default:
			p.log.Info("RPS schedule has been finished. Canceling instance start.")
			cancelStart()
		}
	})
	newInstanceSchedule = func() (core.Schedule, error) {
		return sharedRPSSchedule, err
	}
	return
}

func runNewInstance(ctx context.Context, log *zap.Logger, id string, deps instanceDeps) error {
	instance, err := newInstance(log, id, deps)
	if err != nil {
		return err
	}
	defer instance.Close()
	return instance.Run(ctx)
}

func instanceId(startN int) string {
	return strconv.Itoa(startN)
}

type runResult struct {
	Id  string
	Err error
}

type startResult struct {
	Started int
	Err     error
}

func nonCtxErr(ctx context.Context, err error) bool {
	if err == nil {
		return false
	}
	select {
	case <-ctx.Done():
		if ctx.Err() == errors.Cause(err) { // Support github.com/pkg/errors wrapping
			return false
		}
	default:
	}
	return true
}
