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

	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/gun"
)

// TODO: inject logger
type Base struct {
	Do          func(r *http.Request) (*http.Response, error) // Required.
	Connect     func(ctx context.Context) error               // Optional hook.
	gun.Results                                               // Lazy set via BindResultTo.
}

var _ gun.Gun = (*Base)(nil)

func (b *Base) BindResultsTo(results gun.Results) {
	if b.Results != nil {
		log.Panic("already binded")
	}
	if results == nil {
		log.Panic("nil results")
	}
	b.Results = results
}

// Shoot is thread safe iff Do and Connect hooks are thread safe.
func (b *Base) Shoot(ctx context.Context, a ammo.Ammo) (err error) {
	if b.Results == nil {
		log.Panic("must bind before shoot")
	}
	if b.Connect != nil {
		err = b.Connect(ctx)
		if err != nil {
			log.Printf("Connect error: %s\n", err)
			return
		}
	}

	ha := a.(ammo.HTTP)
	req, sample := ha.Request()
	defer func() {
		if err != nil {
			sample.SetErr(err)
		}
		b.Results <- sample
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
	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		log.Printf("Error reading response body: %s\n", err)
		return
	}
	// TODO: verbose logging
	return
}

func (b *Base) Close() {}
