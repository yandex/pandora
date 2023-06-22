// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package ammo

import (
	"net/http"

	phttp "github.com/yandex/pandora/components/guns/http"
	"github.com/yandex/pandora/core/aggregator/netsample"
)

type Request interface {
	http.Request
}

var _ phttp.Ammo = (*GunAmmo)(nil)

type GunAmmo struct {
	req       *http.Request
	id        uint64
	tag       string
	isInvalid bool
}

func (g GunAmmo) Request() (*http.Request, *netsample.Sample) {
	sample := netsample.Acquire(g.tag)
	sample.SetID(g.id)
	return g.req, sample
}

func (g GunAmmo) ID() uint64 {
	return g.id
}

func (g GunAmmo) IsInvalid() bool {
	return g.isInvalid
}

func NewGunAmmo(req *http.Request, tag string, id uint64) GunAmmo {
	return GunAmmo{
		req: req,
		id:  id,
		tag: tag,
	}
}
