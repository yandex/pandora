package provider

import (
	"context"
	"errors"
	"fmt"
	"net/http"
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
	Sink     chan *base.Ammo[http.Request]
}

func (p *Provider) Acquire() (core.Ammo, bool) {
	ammo, ok := <-p.Sink
	if ok {
		ammo.SetID(p.NextID())
	}
	for _, mw := range p.Middlewares {
		err := mw.UpdateRequest(ammo.Req)
		if err != nil {
			p.Log.Error("error on Middleware.UpdateRequest", zap.Error(err))
			return ammo, false
		}
	}
	return ammo, ok
}

func (p *Provider) Release(a core.Ammo) {
	ammo := a.(*base.Ammo[http.Request])
	// TODO: add request release for example for future fasthttp
	// ammo.Req.Body = nil
	ammo.Req = nil
	p.AmmoPool.Put(ammo)
}

func (p *Provider) Run(ctx context.Context, deps core.ProviderDeps) (err error) {
	var req *http.Request
	var tag string

	p.ProviderDeps = deps
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
		for p.Decoder.Scan(ctx) {
			req, tag = p.Decoder.Next()
			if !confutil.IsChosenCase(tag, p.Config.ChosenCases) {
				continue
			}
			a := p.AmmoPool.Get().(*base.Ammo[http.Request])
			a.Reset(req, tag)
			select {
			case <-ctx.Done():
				err = ctx.Err()
				if err != nil && !errors.Is(err, context.Canceled) {
					err = xerrors.Errorf("error from context: %w", err)
				}
			case p.Sink <- a:
			}
		}

		err = p.Decoder.Err()
		if err != nil {
			if errors.Is(err, decoders.ErrAmmoLimit) || errors.Is(err, decoders.ErrPassLimit) {
				err = nil
			}
			return
		}

		err = ctx.Err()
		if err != nil {
			if !errors.Is(err, context.Canceled) {
				err = xerrors.Errorf("error from context: %w", err)
			}
			return
		}
	}
}

var _ core.Provider = (*Provider)(nil)
