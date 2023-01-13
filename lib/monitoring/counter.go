// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package monitoring

import (
	"expvar"
	"strconv"

	"go.uber.org/atomic"
)

// TODO: use one rcrowley/go-metrics instead.

type Counter struct {
	i atomic.Int64
}

var _ expvar.Var = (*Counter)(nil)

func (c *Counter) String() string {
	return strconv.FormatInt(c.i.Load(), 10)
}

func (c *Counter) Add(delta int64) {
	c.i.Add(delta)
}

func (c *Counter) Set(value int64) {
	c.i.Store(value)
}

func (c *Counter) Get() int64 {
	return c.i.Load()
}

func NewCounter(name string) *Counter {
	v := &Counter{}
	expvar.Publish(name, v)
	return v
}
