package ammo

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/pkg/errors"
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
		Close:    func() {},
	}
}

type Provider struct {
	fs        afero.Fs
	fileName  string
	start     func(ctx context.Context, file afero.File) error
	Sink      chan *Ammo
	Pool      sync.Pool
	idCounter atomic.Uint64
	Close     func()
	core.ProviderDeps
}

func (p *Provider) Acquire() (core.Ammo, bool) {
	ammo, ok := <-p.Sink
	if ok {
		ammo.SetID(p.idCounter.Add(1))
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
	return p.start(ctx, file)
}
