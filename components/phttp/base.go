// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/facebookgo/stackerr"

	"github.com/yandex/pandora/core/aggregate/netsample"
)

// TODO: inject logger
type Base struct {
	Do         func(r *http.Request) (*http.Response, error) // Required.
	Connect    func(ctx context.Context) error               // Optional hook.
	Aggregator netsample.Aggregator                          // Lazy set via BindResultTo.
}

var _ Gun = (*Base)(nil)

func (b *Base) Bind(aggregator netsample.Aggregator) {
	if b.Aggregator != nil {
		log.Panic("already binded")
	}
	if aggregator == nil {
		log.Panic("nil aggregator")
	}
	b.Aggregator = aggregator
}

// Shoot is thread safe iff Do and Connect hooks are thread safe.
func (b *Base) Shoot(ctx context.Context, ammo Ammo) (err error) {
	if b.Aggregator == nil {
		log.Panic("must bind before shoot")
	}
	if b.Connect != nil {
		err = b.Connect(ctx)
		if err != nil {
			log.Printf("Connect error: %s\n", err)
			return
		}
	}

	req, sample := ammo.Request()
	defer func() {
		if err != nil {
			sample.SetErr(err)
		}
		b.Aggregator.Release(sample)
		err = stackerr.WrapSkip(err, 1)
	}()
	var res *http.Response
	res, err = b.Do(req)
	if err != nil {
		log.Printf("Error performing a request: %s\n", err)
		return
	}
	sample.SetProtoCode(res.StatusCode)
	defer res.Body.Close()
	// TODO: measure body read time
	// TODO: buffer copy buffers.
	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		log.Printf("Error reading response body: %s\n", err)
		return
	}
	// TODO: verbose logging
	return
}
