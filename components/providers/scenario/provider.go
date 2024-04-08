package scenario

import (
	"context"
	"errors"
	"fmt"

	"github.com/yandex/pandora/components/providers/base"
	"github.com/yandex/pandora/components/providers/http/decoders"
	"github.com/yandex/pandora/core"
)

type ProviderConfig struct {
	File            string
	Limit           uint
	Passes          uint
	ContinueOnError bool
	MaxAmmoSize     int
}

type ProvAmmo interface {
	SetID(id uint64)
	Clone() ProvAmmo
}

type Provider[A ProvAmmo] struct {
	base.ProviderBase
	cfg ProviderConfig

	sink  chan A
	ammos []A
}

func (p *Provider[A]) SetConfig(conf ProviderConfig) {
	p.cfg = conf
}

func (p *Provider[A]) SetSink(sink chan A) {
	p.sink = sink
}

func (p *Provider[A]) SetAmmos(ammos []A) {
	p.ammos = ammos
}

func (p *Provider[A]) Run(ctx context.Context, deps core.ProviderDeps) error {
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

func (p *Provider[A]) Acquire() (core.Ammo, bool) {
	ammo, ok := <-p.sink
	if !ok {
		return nil, false
	}
	clone := ammo.Clone()
	clone.SetID(p.NextID())
	return clone, true
}

func (p *Provider[A]) Release(_ core.Ammo) {
}
