// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package provider

import (
	"context"

	"github.com/yandex/pandora/core"
)

// NewNum returns dummy provider, that provides 0, 1 .. n int sequence as ammo.
// May be useful for test or in when Gun don't need ammo.
func NewNum(limit int) core.Provider {
	return &num{
		limit: limit,
		sink:  make(chan core.Ammo),
	}
}

type NumConfig struct {
	Limit int
}

func NewNumConf(conf NumConfig) core.Provider {
	return NewNum(conf.Limit)
}

type num struct {
	i     int
	limit int
	sink  chan core.Ammo
}

func (n *num) Run(ctx context.Context) error {
	defer close(n.sink)
	for ; n.limit <= 0 || n.i < n.limit; n.i++ {
		select {
		case n.sink <- n.i:
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

func (n *num) Acquire() (a core.Ammo, ok bool) {
	a, ok = <-n.sink
	return
}

func (n *num) Release(core.Ammo) {}
