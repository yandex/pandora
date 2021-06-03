// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package provider

import (
	"sync"

	"a.yandex-team.ru/load/projects/pandora/core"
)

type AmmoQueueConfig struct {
	// AmmoQueueSize is number maximum number of ready but not acquired ammo.
	// On queue overflow, ammo decode is stopped.
	AmmoQueueSize int `config:"ammo-queue-size" validate:"min=1"`
}

const (
	shootsPerSecondUpperBound = 128 * 1024
	DefaultAmmoQueueSize      = shootsPerSecondUpperBound / 16
)

func DefaultAmmoQueueConfig() AmmoQueueConfig {
	return AmmoQueueConfig{
		AmmoQueueSize: DefaultAmmoQueueSize,
	}
}

func NewAmmoQueue(newAmmo func() core.Ammo, conf AmmoQueueConfig) *AmmoQueue {
	return &AmmoQueue{
		OutQueue: make(chan core.Ammo, conf.AmmoQueueSize),
		InputPool: sync.Pool{New: func() interface{} {
			return newAmmo()
		}},
	}
}

type AmmoQueue struct {
	OutQueue  chan core.Ammo
	InputPool sync.Pool
}

func (p *AmmoQueue) Acquire() (core.Ammo, bool) {
	ammo, ok := <-p.OutQueue
	return ammo, ok
}

func (p *AmmoQueue) Release(a core.Ammo) {
	p.InputPool.Put(a)
}
