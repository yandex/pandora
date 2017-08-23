// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core/aggregate/netsample"
)

type Base struct {
	Log        *zap.Logger                                   // If nil, zap.L() will be used.
	Do         func(r *http.Request) (*http.Response, error) // Required.
	Connect    func(ctx context.Context) error               // Optional hook.
	OnClose    func() error                                  // Optional. Called on Close().
	Aggregator netsample.Aggregator                          // Lazy set via BindResultTo.
}

var _ Gun = (*Base)(nil)
var _ io.Closer = (*Base)(nil)

// TODO(skipor): pass logger here in https://github.com/yandex/pandora/issues/57
func (b *Base) Bind(aggregator netsample.Aggregator) {
	if b.Log == nil {
		b.Log = zap.L()
	}
	if b.Aggregator != nil {
		b.Log.Panic("already binded")
	}
	if aggregator == nil {
		b.Log.Panic("nil aggregator")
	}
	b.Aggregator = aggregator
}

// Shoot is thread safe iff Do and Connect hooks are thread safe.
func (b *Base) Shoot(ctx context.Context, ammo Ammo) {
	if b.Aggregator == nil {
		zap.L().Panic("must bind before shoot")
	}
	if b.Connect != nil {
		err := b.Connect(ctx)
		if err != nil {
			b.Log.Warn("Connect fail", zap.Error(err))
			return
		}
	}

	req, sample := ammo.Request()
	var err error
	defer func() {
		if err != nil {
			sample.SetErr(err)
		}
		b.Aggregator.Report(sample)
		err = errors.WithStack(err)
	}()
	if ent := b.Log.Check(zap.DebugLevel, "Shoot"); ent != nil {
		ent.Write(zap.Stringer("url", req.URL))
	}
	var res *http.Response
	res, err = b.Do(req)
	if err != nil {
		b.Log.Warn("Request fail", zap.Error(err))
		return
	}
	if ent := b.Log.Check(zap.DebugLevel, "Got response"); ent != nil {
		ent.Write(zap.Int("status", res.StatusCode))
	}
	sample.SetProtoCode(res.StatusCode)
	defer res.Body.Close()
	// TODO: measure body read time
	_, err = io.Copy(ioutil.Discard, res.Body) // Buffers are pooled for ioutil.Discard
	if err != nil {
		b.Log.Warn("Body read fail", zap.Error(err))
		return
	}
	// TODO: verbose logging
}

func (b *Base) Close() error {
	if b.OnClose != nil {
		return b.OnClose()
	}
	return nil
}
