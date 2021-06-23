// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package provider

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/errutil"
	"github.com/yandex/pandora/lib/ioutil2"
)

type NewAmmoDecoder func(deps core.ProviderDeps, source io.Reader) (AmmoDecoder, error)

// TODO(skipo): test decoder that fills ammo with random data

// AmmoEncoder MAY support only concrete type of ammo.
// AmmoDecoder SHOULD NOT be used after first decode fail.
type AmmoDecoder interface {
	// Decode fills passed ammo with data.
	// Returns non nil error on fail.
	// Panics if ammo type is not supported.
	Decode(ammo core.Ammo) error
}

type AmmoDecoderFunc func(ammo core.Ammo) error

func (f AmmoDecoderFunc) Decode(ammo core.Ammo) error { return f(ammo) }

// TODO(skipor): test

func NewDecodeProvider(newAmmo func() core.Ammo, newDecoder NewAmmoDecoder, conf DecodeProviderConfig) *DecodeProvider {
	return &DecodeProvider{
		AmmoQueue:  *NewAmmoQueue(newAmmo, conf.Queue),
		newDecoder: newDecoder,
		conf:       conf,
	}
}

type DecodeProviderConfig struct {
	Queue  AmmoQueueConfig `config:",squash"`
	Source core.DataSource `config:"source" validate:"required"`
	// Limit limits total num of ammo. Unlimited if zero.
	Limit int `validate:"min=0"`
	// Passes limits ammo file passes. Unlimited if zero.
	Passes int `validate:"min=0"`
}

func DefaultDecodeProviderConfig() DecodeProviderConfig {
	return DecodeProviderConfig{
		Queue: DefaultAmmoQueueConfig(),
	}
}

type DecodeProvider struct {
	AmmoQueue
	conf       DecodeProviderConfig
	newDecoder NewAmmoDecoder
	core.ProviderDeps
}

var _ core.Provider = &DecodeProvider{}

func (p *DecodeProvider) Run(ctx context.Context, deps core.ProviderDeps) (err error) {
	p.ProviderDeps = deps
	defer close(p.OutQueue)
	source, err := p.conf.Source.OpenSource()
	if err != nil {
		return errors.WithMessage(err, "data source open failed")
	}
	defer func() {
		_ = errutil.Join(err, errors.Wrap(source.Close(), "data source close failed"))
	}()

	// Problem: can't use decoder after io.EOF, because decoder is invalidated. But decoder recreation
	// is not efficient, when we have short data source.
	// Now problem solved by using MultiPassReader, but in such case decoder don't know real input
	// position, so can't put this important information in decode error.
	// TODO(skipor):  Let's add optional Reset(io.Reader) method, that will allow efficient Decoder reset after every pass.
	multipassReader := ioutil2.NewMultiPassReader(source, p.conf.Passes)
	if source == multipassReader {
		p.Log.Info("Ammo data source can't sought, so will be read only once")
	}
	decoder, err := p.newDecoder(deps, multipassReader)

	if err != nil {
		return errors.WithMessage(err, "decoder construction failed")
	}
	var ammoNum int
	for ; p.conf.Limit <= 0 || ammoNum < p.conf.Limit; ammoNum++ {
		ammo := p.InputPool.Get()
		err = decoder.Decode(ammo)
		if err == io.EOF {
			p.Log.Info("Ammo finished", zap.Int("decoded", ammoNum))
			return nil
		}
		if err != nil {
			return errors.WithMessage(err, fmt.Sprintf("ammo #%v decode failed", ammoNum))
		}
		select {
		case p.OutQueue <- ammo:
		case <-ctx.Done():
			p.Log.Debug("Provider run context is Done", zap.Int("decoded", ammoNum+1))
			return nil
		}
	}
	p.Log.Info("Ammo limit is reached", zap.Int("decoded", ammoNum))
	return nil
}
