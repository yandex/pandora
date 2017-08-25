// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package engine

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coreutil"
)

type instance struct {
	log *zap.Logger
	id  string
	instanceDeps
}

func newInstance(log *zap.Logger, id string, deps instanceDeps) *instance {
	log = log.With(zap.String("instance", id))
	return &instance{log, id, deps}
}

type instanceDeps struct {
	provider    core.Provider
	aggregator  core.Aggregator
	newSchedule func() (core.Schedule, error)
	newGun      func() (core.Gun, error)
	Metrics
}

// Run blocks until ammo finish, error or context cancel.
func (i *instance) Run(ctx context.Context) error {
	// Creating deps in instance start, which is running in separate goroutine.
	// That allows to create instances parallel and faster.

	i.log.Debug("Instance started")
	i.InstanceStart.Add(1)
	defer func() {
		defer i.log.Debug("Instance finished")
		i.InstanceFinish.Add(1)
	}()

	shed, err := i.newSchedule()
	if err != nil {
		return fmt.Errorf("schedule create failed: %s", err)
	}
	gun, err := i.newGun()
	if err != nil {
		return fmt.Errorf("gun create failed: %s", err)
	}

	if gun, ok := gun.(io.Closer); ok {
		defer func() {
			err := gun.Close()
			if err != nil {
				i.log.Warn("Gun close fail", zap.Error(err))
			}
			i.log.Debug("Gun closed")
		}()
	}

	gun.Bind(i.aggregator)
	nextShoot := coreutil.NewWaiter(shed, ctx)

	i.log.Debug("Instance init done. Run shooting")
	return i.shoot(ctx, gun, nextShoot)
}

func (i *instance) shoot(ctx context.Context, gun core.Gun, next *coreutil.Waiter) (err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = errors.Errorf("shoot panic: %s", r)
		}
	}()
	for {
		// Try get ammo before schedule wait, to be ready shoot just in time.
		// Acquire should unblock in case of context cancel.
		// TODO: we just throw away acquired ammo, if our schedule finished. Fix it.
		ammo, more := i.provider.Acquire()
		if !more {
			i.log.Debug("Ammo ended")
			break
		}
		if !next.Wait() {
			break
		}
		i.Request.Add(1)
		gun.Shoot(ctx, ammo)
		i.Response.Add(1)
		i.provider.Release(ammo)
	}
	return ctx.Err()
}
