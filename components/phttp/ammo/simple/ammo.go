// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package simple

import (
	"net/http"

	"github.com/yandex/pandora/components/phttp"
	"github.com/yandex/pandora/core/aggregate/netsample"
)

type Ammo struct {
	// OPTIMIZE(skipor): reuse *http.Request.
	// Need to research is it possible. http.Transport can hold reference to http.Request.
	req *http.Request
	tag string
}

func NewAmmo(req *http.Request, tag string) *Ammo {
	return &Ammo{req, tag}
}

func (a *Ammo) Request() (*http.Request, *netsample.Sample) {
	sample := netsample.Acquire(a.tag)
	return a.req, sample
}

func (a *Ammo) Reset(req *http.Request, tag string) {
	*a = Ammo{req, tag}
}

var _ phttp.Ammo = (*Ammo)(nil)
