// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package ammo

import (
	"net/http"

	"github.com/yandex/pandora/components/phttp"
	"github.com/yandex/pandora/core/aggregate"
)

type Simple struct {
	// OPTIMIZE(skipor): reuse *http.Request.
	// Need to research is it possible. http.Transport can hold reference to http.Request.
	req *http.Request
	tag string
}

func NewSimpleHTTP(req *http.Request, tag string) *Simple {
	return &Simple{req, tag}
}

func (a *Simple) Request() (*http.Request, *aggregate.Sample) {
	sample := aggregate.AcquireSample(a.tag)
	return a.req, sample
}

func (a *Simple) Reset(req *http.Request, tag string) {
	*a = Simple{req, tag}
}

var _ phttp.Ammo = (*Simple)(nil)
