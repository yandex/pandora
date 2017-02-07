// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package ammo

import (
	"io"
	"net/http"

	"github.com/yandex/pandora/aggregate"
)

// HTTP ammo interface for http based guns.
// HTTP ammo providers should produce ammo that implements HTTP.
// HTTP guns should use convert ammo to HTTP interface, not to specific implementation.
type HTTP interface {
	// TODO (skipor) instead of sample use some more usable interface.
	Request() (*http.Request, *aggregate.Sample)
}

type SimpleHTTP struct {
	req *http.Request
	tag string
}

func (a *SimpleHTTP) Request() (*http.Request, *aggregate.Sample) {
	sample := aggregate.AcquireSample(a.tag)
	return a.req, sample
}

func (a *SimpleHTTP) Reset(req *http.Request, tag string) {
	*a = SimpleHTTP{req, tag}
}

var _ HTTP = (*SimpleHTTP)(nil)

type ReadSeeker interface {
	io.ReadSeeker
	io.ReaderAt
}

// TODO (skipor): need it?
type HTTPDecoder interface {
	Decode(r *http.Request) error
	Reset() error
}
