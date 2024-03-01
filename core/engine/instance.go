package engine

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/coreutil"
	"github.com/yandex/pandora/core/warmup"
	"github.com/yandex/pandora/lib/tag"
	"go.uber.org/zap"
)

type instance struct {
	log      *zap.Logger
	id       int
	gun      core.Gun
	schedule core.Schedule
	instanceSharedDeps
}

func newInstance(ctx context.Context, log *zap.Logger, poolID string, id int, deps instanceDeps) (*instance, error) {
	log = log.With(zap.Int("instance", id))
	gunDeps := core.GunDeps{Ctx: ctx, Log: log, PoolID: poolID, InstanceID: id}
	sched, err := deps.newSchedule()
	if err != nil {
		return nil, err
	}
	gun, err := deps.newGun()
	if err != nil {
		return nil, err
	}
	if warmedUp, ok := gun.(warmup.WarmedUp); ok {
		if err := warmedUp.AcceptWarmUpResult(deps.gunWarmUpResult); err != nil {
			return nil, fmt.Errorf("gun failed to accept warmup result: %w", err)
		}
	}
	err = gun.Bind(deps.aggregator, gunDeps)
	if err != nil {
		return nil, err
	}
	inst := &instance{log, id, gun, sched, deps.instanceSharedDeps}
	return inst, nil
}

type instanceDeps struct {
	newSchedule func() (core.Schedule, error)
	newGun      func() (core.Gun, error)
	instanceSharedDeps
}

type instanceSharedDeps struct {
	provider        core.Provider
	metrics         Metrics
	gunWarmUpResult interface{}
	aggregator      core.Aggregator
	discardOverflow bool
}

// Run blocks until ammo finish, error or context cancel.
// Expects, that gun is already bind.
func (i *instance) Run(ctx context.Context) (recoverErr error) {
	defer func() {
		r := recover()
		if r != nil {
			recoverErr = errors.Errorf("shoot panic: %s", r)
		}

		i.log.Debug("Instance finished")
		i.metrics.InstanceFinish.Add(1)
	}()
	i.log.Debug("Instance started")
	i.metrics.InstanceStart.Add(1)

	waiter := coreutil.NewWaiter(i.schedule)
	// Checking, that schedule is not finished, required, to not consume extra ammo,
	// on finish in case of per instance schedule.
	for !waiter.IsFinished(ctx) {
		err := func() error {
			ammo, ok := i.provider.Acquire()
			if !ok {
				i.log.Debug("Out of ammo")
				return outOfAmmoErr
			}
			defer i.provider.Release(ammo)
			if tag.Debug {
				i.log.Debug("Ammo acquired", zap.Any("ammo", ammo))
			}
			if !waiter.Wait(ctx) {
				return nil
			}
			if !i.discardOverflow || !waiter.IsSlowDown(ctx) {
				i.metrics.Request.Add(1)
				if tag.Debug {
					i.log.Debug("Shooting", zap.Any("ammo", ammo))
				}
				i.gun.Shoot(ammo)
				i.metrics.Response.Add(1)
			} else {
				i.aggregator.Report(netsample.DiscardedShootSample())
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return ctx.Err()
}

func (i *instance) Close() error {
	gunCloser, ok := i.gun.(io.Closer)
	if !ok {
		return nil
	}
	err := gunCloser.Close()
	if err != nil {
		i.log.Warn("Gun close fail", zap.Error(err))
	}
	i.log.Debug("Gun closed")
	return err
}

var outOfAmmoErr = errors.New("Out of ammo")
