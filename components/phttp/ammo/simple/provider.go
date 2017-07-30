// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package simple

import (
	"context"
	"sync"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/afero"
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
	fs       afero.Fs
	fileName string
	start    func(ctx context.Context, file afero.File) error
	Sink     chan *Ammo
	Pool     sync.Pool
}

func (p *Provider) Acquire() (ammo core.Ammo, ok bool) {
	ammo, ok = <-p.Sink
	return
}

func (p *Provider) Release(a core.Ammo) {
	p.Pool.Put(a)
}

func (p *Provider) Run(ctx context.Context) error {
	defer close(p.Sink)
	file, err := p.fs.Open(p.fileName)
	if err != nil {
		// TODO(skipor): instead of passing stacktrace log error here.
		return stackerr.Newf("failed to open ammo Sink: %v", err)
	}
	defer file.Close()
	return p.start(ctx, file)
}
