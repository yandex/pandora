package scenario

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/afero"

	"github.com/yandex/pandora/components/providers/base"
	"github.com/yandex/pandora/components/providers/http/decoders"
	"github.com/yandex/pandora/core"
)

const defaultSinkSize = 100

func NewProvider(fs afero.Fs, conf Config) (core.Provider, error) {
	const op = "scenario.NewProvider"
	if conf.File == "" {
		return nil, fmt.Errorf("scenario provider config should contain non-empty 'file' field")
	}
	file, err := fs.Open(conf.File)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}
	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			if err != nil {
				err = fmt.Errorf("%s multiple errors faced: %w, with close err: %s", op, err, closeErr)
			} else {
				err = fmt.Errorf("%s, %w", op, closeErr)
			}
		}
	}()

	d := &decoder{}
	ammos, err := d.parseAmmo(file, conf)
	if err != nil {
		return nil, fmt.Errorf("%s %w", op, err)
	}

	return &Provider{
		cfg:   conf,
		sink:  make(chan *Ammo, defaultSinkSize),
		ammos: ammos,
	}, nil
}

type Provider struct {
	base.ProviderBase
	cfg Config

	sink  chan *Ammo
	ammos []*Ammo
}

func (p *Provider) Run(ctx context.Context, deps core.ProviderDeps) error {
	const op = "scenario.Provider.Run"
	p.Deps = deps

	length := uint(len(p.ammos))
	if length == 0 {
		return decoders.ErrNoAmmo
	}
	ammoNum := uint(0)
	passNum := uint(0)
	for {
		err := ctx.Err()
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				err = fmt.Errorf("%s error from context: %w", op, err)
			}
			return err
		}
		i := ammoNum % length
		passNum = ammoNum / length
		if p.cfg.Passes != 0 && passNum >= p.cfg.Passes {
			return decoders.ErrPassLimit
		}
		if p.cfg.Limit != 0 && ammoNum >= p.cfg.Limit {
			return decoders.ErrAmmoLimit
		}
		ammoNum++
		ammo := p.ammos[i]
		select {
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil && !errors.Is(err, context.Canceled) {
				err = fmt.Errorf("%s error from context: %w", op, err)
			}
			return err
		case p.sink <- ammo:
		}
	}
}

func (p *Provider) Acquire() (core.Ammo, bool) {
	ammo, ok := <-p.sink
	if !ok {
		return nil, false
	}
	return ammo, true
}

func (p *Provider) Release(_ core.Ammo) {
	return
}

var _ core.Provider = (*Provider)(nil)
