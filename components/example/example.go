// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package example

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregate/netsample"
)

type Ammo struct {
	Message string
}

func NewGun() *Gun {
	return &Gun{}
}

type Gun struct {
	aggregator core.Aggregator
	core.GunDeps
}

var _ core.Gun = &Gun{}

func (l *Gun) Bind(aggregator core.Aggregator, deps core.GunDeps) error {
	l.aggregator = aggregator
	l.GunDeps = deps
	return nil
}

func (l *Gun) Shoot(a core.Ammo) {
	sample := netsample.Acquire("REQUEST")
	// Do work here.
	l.Log.Info("Example Gun message", zap.String("message", a.(*Ammo).Message))
	sample.SetProtoCode(200)
	l.aggregator.Report(sample)
}

type ProviderConfig struct {
	AmmoLimit int `config:"limit"`
}

func NewDefaultProviderConfig() ProviderConfig {
	return ProviderConfig{AmmoLimit: 16}
}

func NewProvider(conf ProviderConfig) *Provider {
	return &Provider{
		ProviderConfig: conf,
		sink:           make(chan *Ammo, 128),
		pool:           sync.Pool{New: func() interface{} { return &Ammo{} }},
	}
}

type Provider struct {
	ProviderConfig
	sink chan *Ammo
	pool sync.Pool
}

var _ core.Provider = &Provider{}

func (p *Provider) Acquire() (ammo core.Ammo, ok bool) {
	ammo, ok = <-p.sink
	return
}

func (p *Provider) Release(ammo core.Ammo) {
	p.pool.Put(ammo)
}

func (p *Provider) Run(ctx context.Context, deps core.ProviderDeps) error {
	defer close(p.sink)
	for i := 0; i < p.AmmoLimit; i++ {
		select {
		case p.sink <- p.newAmmo(i):
			continue
		case <-ctx.Done():
		}
		break
	}
	return nil
}

func (p *Provider) newAmmo(i int) *Ammo {
	ammo := p.pool.Get().(*Ammo)
	ammo.Message = fmt.Sprintf(`Job #%d"`, i)
	return ammo
}
