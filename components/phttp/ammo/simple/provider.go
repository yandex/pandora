// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package simple

import (
	"context"
	"sync"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/atomic"

	"github.com/yandex/pandora/core"
)

func NewProvider(fs afero.Fs, fileName string, start func(ctx context.Context, file afero.File) error) Provider {
	return Provider{
		fs:       fs,
		fileName: fileName,
		start:    start,
		Sink:     make(chan *Ammo, 128),
		Pool:     sync.Pool{New: func() interface{} { return &Ammo{} }},
	}
}

type Provider struct {
	fs        afero.Fs
	fileName  string
	start     func(ctx context.Context, file afero.File) error
	Sink      chan *Ammo
	Pool      sync.Pool
	idCounter atomic.Int64
	core.ProviderDeps
}

func (p *Provider) Acquire() (core.Ammo, bool) {
	ammo, ok := <-p.Sink
	if ok {
		ammo.SetID(int(p.idCounter.Inc() - 1))
	}
	return ammo, ok
}

func (p *Provider) Release(a core.Ammo) {
	p.Pool.Put(a)
}

func (p *Provider) Run(ctx context.Context, deps core.ProviderDeps) error {
	p.ProviderDeps = deps
	defer close(p.Sink)
	file, err := p.fs.Open(p.fileName)
	if err != nil {
		return errors.Wrap(err, "failed to open ammo file")
	}
	defer file.Close()
	return p.start(ctx, file)
}
