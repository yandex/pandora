// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package engine

import (
	"context"
	"io"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coreutil"
	"github.com/yandex/pandora/lib/tag"
)

type instance struct {
	log      *zap.Logger
	id       int
	gun      core.Gun
	schedule core.Schedule
	instanceSharedDeps
}

func newInstance(ctx context.Context, log *zap.Logger, id int, deps instanceDeps) (*instance, error) {
	log = log.With(zap.Int("instance", id))
	gunDeps := core.GunDeps{Ctx: ctx, Log: log, InstanceID: id}
	sched, err := deps.newSchedule()
	if err != nil {
		return nil, err
	}
	gun, err := deps.newGun()
	if err != nil {
		return nil, err
	}
	err = gun.Bind(deps.aggregator, gunDeps)
	if err != nil {
		return nil, err
	}
	inst := &instance{log, id, gun, sched, deps.instanceSharedDeps}
	return inst, nil
}

type instanceDeps struct {
	aggregator  core.Aggregator
	newSchedule func() (core.Schedule, error)
	newGun      func() (core.Gun, error)
	instanceSharedDeps
}

type instanceSharedDeps struct {
	provider core.Provider
	Metrics
}

// Run blocks until ammo finish, error or context cancel.
// Expects, that gun is already bind.
func (i *instance) Run(ctx context.Context) error {
	i.log.Debug("Instance started")
	i.InstanceStart.Add(1)
	defer func() {
		defer i.log.Debug("Instance finished")
		i.InstanceFinish.Add(1)
	}()

	return i.shoot(ctx)
}

func (i *instance) shoot(ctx context.Context) (err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = errors.Errorf("shoot panic: %s", r)
		}
	}()

	waiter := coreutil.NewWaiter(i.schedule, ctx)
	// Checking, that schedule is not finished, required, to not consume extra ammo,
	// on finish in case of per instance schedule.
	for !waiter.IsFinished() {
		ammo, ok := i.provider.Acquire()
		if !ok {
			i.log.Debug("Out of ammo")
			break
		}
		if tag.Debug {
			i.log.Debug("Ammo acquired", zap.Any("ammo", ammo))
		}
		if !waiter.Wait() {
			break
		}
		i.Metrics.Request.Add(1)
		if tag.Debug {
			i.log.Debug("Shooting", zap.Any("ammo", ammo))
		}
		i.gun.Shoot(ammo)
		i.Metrics.Response.Add(1)
		i.provider.Release(ammo)
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
