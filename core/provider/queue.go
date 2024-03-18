package provider

import (
	"sync"

	"github.com/yandex/pandora/core"
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
