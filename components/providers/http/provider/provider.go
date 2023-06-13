package provider

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/yandex/pandora/components/providers/base"
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

	AmmoPool sync.Pool
	Sink     chan *base.Ammo
}

func (p *Provider) Acquire() (core.Ammo, bool) {
	ammo, ok := <-p.Sink
	if ok {
		ammo.SetID(p.NextID())
	}
	if err := ammo.BuildRequest(); err != nil {
		p.Deps.Log.Error("http build request error", zap.Error(err))
		return ammo, false
	}
	for _, mw := range p.Middlewares {
		err := mw.UpdateRequest(ammo.Req)
		if err != nil {
			p.Deps.Log.Error("error on Middleware.UpdateRequest", zap.Error(err))
			return ammo, false
		}
	}
	return ammo, ok
}

func (p *Provider) Release(a core.Ammo) {
	ammo := a.(*base.Ammo)
	ammo.Reset()
	p.AmmoPool.Put(ammo)
}

func (p *Provider) Run(ctx context.Context, deps core.ProviderDeps) (err error) {
	var ammo *base.Ammo

	p.Deps = deps
	defer func() {
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

	for {
		if err = ctx.Err(); err != nil {
			if !errors.Is(err, context.Canceled) {
				err = xerrors.Errorf("error from context: %w", err)
			}
			return
		}
		ammo, err = p.Decoder.Scan(ctx)
		if !confutil.IsChosenCase(ammo.Tag(), p.Config.ChosenCases) {
			continue
		}
		if err != nil {
			if errors.Is(err, decoders.ErrAmmoLimit) || errors.Is(err, decoders.ErrPassLimit) {
				err = nil
			}
			return
		}

		select {
		case <-ctx.Done():
			err = ctx.Err()
			if err != nil && !errors.Is(err, context.Canceled) {
				err = xerrors.Errorf("error from context: %w", err)
			}
			return
		case p.Sink <- ammo:
		}
	}
}

var _ core.Provider = (*Provider)(nil)
