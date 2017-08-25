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
	"net/url"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core/aggregate/netsample"
)

const (
	EmptyTag = "__EMPTY__"
)

type BaseGunConfig struct {
	AutoTag AutoTagConfig `config:"auto-tag"`
}

// AutoTagConfig configure automatic tags generation based on ammo URI. First AutoTag URI path elements becomes tag.
// Example: /my/very/deep/page?id=23&param=33 -> /my/very when uri-elements: 2.
type AutoTagConfig struct {
	Enabled     bool `config:"enabled"`
	URIElements int  `config:"uri-elements" validate:"min=1"` // URI elements used to autotagging
	NoTagOnly   bool `config:"no-tag-only"`                   // When true, autotagged only ammo that has no tag before.
}

func NewDefaultBaseGunConfig() BaseGunConfig {
	return BaseGunConfig{
		AutoTagConfig{
			Enabled:     false,
			URIElements: 2,
			NoTagOnly:   true,
		}}
}

type BaseGun struct {
	Log        *zap.Logger // If nil, zap.L() will be used.
	DebugLog   bool        // Automaticaly set in Bind if Log accepts debug messages.
	Config     BaseGunConfig
	Do         func(r *http.Request) (*http.Response, error) // Required.
	Connect    func(ctx context.Context) error               // Optional hook.
	OnClose    func() error                                  // Optional. Called on Close().
	Aggregator netsample.Aggregator                          // Lazy set via BindResultTo.
}

var _ Gun = (*BaseGun)(nil)
var _ io.Closer = (*BaseGun)(nil)

// TODO(skipor): pass logger here in https://github.com/yandex/pandora/issues/57
func (b *BaseGun) Bind(aggregator netsample.Aggregator) {
	if b.Log == nil {
		b.Log = zap.L()
	}
	if ent := b.Log.Check(zap.DebugLevel, "Gun bind"); ent != nil {
		// Enable debug level logging during shooting. Creating log entries isn't free.
		b.DebugLog = true
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
func (b *BaseGun) Shoot(ctx context.Context, ammo Ammo) {
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
	if b.DebugLog {
		b.Log.Debug("Shoot", zap.Stringer("url", req.URL))
	}

	if b.Config.AutoTag.Enabled && (!b.Config.AutoTag.NoTagOnly || sample.Tags() == "") {
		sample.AddTag(autotag(b.Config.AutoTag.URIElements, req.URL))
	}
	if sample.Tags() == "" {
		sample.AddTag(EmptyTag)
	}

	var err error
	defer func() {
		if err != nil {
			sample.SetErr(err)
		}
		b.Aggregator.Report(sample)
		err = errors.WithStack(err)
	}()

	var res *http.Response
	res, err = b.Do(req)
	if err != nil {
		b.Log.Warn("Request fail", zap.Error(err))
		return
	}
	if b.DebugLog {
		b.Log.Debug("Got response", zap.Int("status", res.StatusCode))
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

func (b *BaseGun) Close() error {
	if b.OnClose != nil {
		return b.OnClose()
	}
	return nil
}

func autotag(depth int, URL *url.URL) string {
	path := URL.Path
	var ind int
	for ; ind < len(path); ind++ {
		if path[ind] == '/' {
			if depth == 0 {
				break
			}
			depth--
		}
	}
	return path[:ind]
}
