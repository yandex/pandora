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
				return fmt.Errorf("%q pool run failed: %s", res.Id, res.Err)
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
	originalCtx := ctx                     // Canceled only in case of other pool fail.
	ctx, cancel := context.WithCancel(ctx) // Canceled in case of fail, or all instances finish.

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
		// Running in separate goroutine, so we can cancel instance creation in case of error.
		started, err := p.startInstances(ctx, runRes)
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
			startedInstances = -1
			awaitedInstances = 0
		)
		onAllInstancesFinished := func() {
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
					errAwaited(fmt.Errorf("provider failed: %s", err))
				}
			case err := <-aggregatorErr:
				aggregatorErr = nil
				toWait--
				p.log.Debug("Aggregator awaited", zap.Error(err))
				if nonCtxErr(ctx, err) {
					errAwaited(fmt.Errorf("aggregator failed: %s", err))
				}
			case res := <-startRes:
				startRes = nil
				toWait--
				startedInstances = res.Started
				p.log.Debug("Instances start awaited", zap.Int("started", startedInstances), zap.Error(res.Err))
				if res.Err != nil {
					errAwaited(fmt.Errorf("instances start failed: %s", res.Err))
				}
				if startedInstances <= awaitedInstances {
					onAllInstancesFinished()
				}
			case res := <-runRes:
				awaitedInstances++
				if ent := p.log.Check(zap.DebugLevel, "Instance run awaited"); ent != nil {
					ent.Write(zap.String("id", res.Id), zap.Error(res.Err), zap.Int("awaited", awaitedInstances))
				}
				if res.Err != nil {
					errAwaited(fmt.Errorf("istance %q run failed: %s", res.Id, res.Err))
				}
				startFinished := startRes == nil
				if !startFinished || awaitedInstances < startedInstances {
					continue
				}
				onAllInstancesFinished()
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

func (p *instancePool) startInstances(ctx context.Context, runRes chan<- runResult) (started int, err error) {
	newInstanceSchedule := func() func() (core.Schedule, error) {
		if p.RPSPerInstance {
			return p.NewRPSSchedule
		}
		sharedSchedule, err := p.NewRPSSchedule()
		return func() (core.Schedule, error) {
			return sharedSchedule, err
		}
	}()
	deps := instanceDeps{p.Provider, p.Aggregator, newInstanceSchedule, p.NewGun, p.metrics}
	for next := coreutil.NewWaiter(p.StartupSchedule, ctx); next.Wait(); started++ {
		id := strconv.Itoa(started)
		instance := newInstance(p.log, id, deps)
		go func() {
			runRes <- runResult{instance.id, instance.Run(ctx)}
		}()
	}
	err = ctx.Err()
	return
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
