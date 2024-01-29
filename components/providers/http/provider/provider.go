package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/yandex/pandora/components/providers/base"
	httpProvider "github.com/yandex/pandora/components/providers/http/ammo"
	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/confutil"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

type Provider struct {
	base.ProviderBase
	config.Config
	decoders.Decoder

	Close func() error

	Sink  chan decoders.DecodedAmmo
	ammos []decoders.DecodedAmmo
}

func (p *Provider) Acquire() (core.Ammo, bool) {
	ammo, ok := <-p.Sink
	if !ok {
		return nil, false
	}
	req, err := ammo.BuildRequest()
	if err != nil {
		p.Deps.Log.Error("http build request error", zap.Error(err))
		return ammo, false
	}
	for _, mw := range p.Middlewares {
		err := mw.UpdateRequest(req)
		if err != nil {
			p.Deps.Log.Error("error on Middleware.UpdateRequest", zap.Error(err))
			return ammo, false
		}
	}
	return httpProvider.NewGunAmmo(req, ammo.Tag(), p.NextID()), ok
}

func (p *Provider) Release(a core.Ammo) {
	if p.Preload {
		return
	}
	p.Decoder.Release(a)
}

func (p *Provider) Run(ctx context.Context, deps core.ProviderDeps) (err error) {
	p.Deps = deps
	defer func() {
		close(p.Sink)
		// TODO: wrap in go 1.20
		// err = errors.Join(err, p.Close())
		if p.Close == nil {
			return
		}
		closeErr := p.Close()
		if closeErr != nil {
			if err != nil {
				err = xerrors.Errorf("Multiple errors faced: %w, %w", err, closeErr)
			} else {
				err = closeErr
			}
		}
	}()

	for _, mw := range p.Middlewares {
		if err := mw.InitMiddleware(ctx, deps.Log); err != nil {
			return fmt.Errorf("cant InitMiddleware %T, err: %w", mw, err)
		}
	}

	if p.Config.Preload {
		err = p.loadAmmo(ctx)
		if err == nil {
			err = p.runPreloaded(ctx)
		}
	} else {
		err = p.runFullScan(ctx)
	}

	return
}

func (p *Provider) runFullScan(ctx context.Context) error {
	for {
		if err := ctx.Err(); err != nil {
			if !errors.Is(err, context.Canceled) {
				err = xerrors.Errorf("error from context: %w", err)
			}
			return err
		}
		ammo, err := p.Decoder.Scan(ctx)
		if err != nil {
			if errors.Is(err, decoders.ErrAmmoLimit) || errors.Is(err, decoders.ErrPassLimit) {
				err = nil
			}
			return err
		}
		if !confutil.IsChosenCase(ammo.Tag(), p.Config.ChosenCases) {
			continue
		}

		select {
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil && !errors.Is(err, context.Canceled) {
				err = xerrors.Errorf("error from context: %w", err)
			}
			return err
		case p.Sink <- ammo:
		}
	}
}

func (p *Provider) loadAmmo(ctx context.Context) error {
	ammos, err := p.Decoder.LoadAmmo(ctx)
	if err != nil {
		return fmt.Errorf("cant LoadAmmo, err: %w", err)
	}
	p.ammos = make([]decoders.DecodedAmmo, 0, len(ammos))
	for _, ammo := range ammos {
		if confutil.IsChosenCase(ammo.Tag(), p.Config.ChosenCases) {
			p.ammos = append(p.ammos, ammo)
		}
	}
	return nil
}

func (p *Provider) runPreloaded(ctx context.Context) error {
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
				err = xerrors.Errorf("error from context: %w", err)
			}
			return err
		}
		i := ammoNum % length
		passNum = ammoNum / length
		if p.Passes != 0 && passNum >= p.Passes {
			return decoders.ErrPassLimit
		}
		if p.Limit != 0 && ammoNum >= p.Limit {
			return decoders.ErrAmmoLimit
		}
		ammoNum++
		ammo := p.ammos[i]
		select {
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil && !errors.Is(err, context.Canceled) {
				err = xerrors.Errorf("error from context: %w", err)
			}
			return err
		case p.Sink <- ammo:
		}
	}
}

var _ core.Provider = (*Provider)(nil)
