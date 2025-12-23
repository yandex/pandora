package provider

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/providers/grpc/ammo"
	"github.com/yandex/pandora/components/providers/grpc/middleware"
	"github.com/yandex/pandora/core"
	"go.uber.org/zap"
)

func NewProvider(fs afero.Fs, fileName string, start func(ctx context.Context, file afero.File) error, middlewares []middleware.Middleware) Provider {
	return Provider{
		fs:          fs,
		fileName:    fileName,
		start:       start,
		Sink:        make(chan *ammo.Ammo, 128),
		Pool:        sync.Pool{New: func() interface{} { return &ammo.Ammo{} }},
		Close:       func() {},
		Middlewares: middlewares,
	}
}

type Provider struct {
	fs        afero.Fs
	fileName  string
	start     func(ctx context.Context, file afero.File) error
	Sink      chan *ammo.Ammo
	Pool      sync.Pool
	idCounter atomic.Uint64
	Close     func()
	core.ProviderDeps
	Middlewares []middleware.Middleware
}

func (p *Provider) Acquire() (core.Ammo, bool) {
	ammo, ok := <-p.Sink
	if ok {
		ammo.SetID(p.idCounter.Add(1))

		for _, mw := range p.Middlewares {
			err := mw.UpdateRequest(ammo)
			if err != nil {
				p.ProviderDeps.Log.Error("error on Middleware.UpdateRequest", zap.Error(err))
				return ammo, false
			}
		}
	}
	return ammo, ok
}

func (p *Provider) Release(a core.Ammo) {
	p.Pool.Put(a)
}

func (p *Provider) Run(ctx context.Context, deps core.ProviderDeps) error {
	defer p.Close()
	p.ProviderDeps = deps
	defer close(p.Sink)
	file, err := p.fs.Open(p.fileName)
	if err != nil {
		return errors.Wrap(err, "failed to open ammo file")
	}
	defer file.Close()

	for _, mw := range p.Middlewares {
		if err := mw.InitMiddleware(ctx, deps.Log); err != nil {
			return fmt.Errorf("cant InitMiddleware %T, err: %w", mw, err)
		}
	}

	return p.start(ctx, file)
}
