// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package simple

import (
	"net/http"

	"github.com/yandex/pandora/components/phttp"
	"github.com/yandex/pandora/core/aggregator/netsample"
)

type Ammo struct {
	// OPTIMIZE(skipor): reuse *http.Request.
	// Need to research is it possible. http.Transport can hold reference to http.Request.
	req *http.Request
	tag string
	id  int
	isInvalid bool
}

func (a *Ammo) Request() (*http.Request, *netsample.Sample) {
	sample := netsample.Acquire(a.tag)
	sample.SetId(a.id)
	return a.req, sample
}

func (a *Ammo) Reset(req *http.Request, tag string) {
	*a = Ammo{req, tag, -1, false}
}

func (a *Ammo) SetId(id int) {
	a.id = id
}

func (a *Ammo) Id() int {
	return a.id
}

func (a *Ammo) Invalidate() {
	a.isInvalid = true
}

func (a *Ammo) IsInvalid() bool {
	return a.isInvalid
}

func (a *Ammo) IsValid() bool {
	return !a.isInvalid
}

var _ phttp.Ammo = (*Ammo)(nil)
