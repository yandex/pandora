// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package example

import (
	"sync"

	"github.com/yandex/pandora/core"
)

type Decoder interface {
	Decode([]byte, *Ammo) (*Ammo, error)
}

func NewBaseProvider(bufSize int, decoder Decoder, New func() interface{}) *BaseProvider {
	ch := make(chan core.Ammo, bufSize)
	return &BaseProvider{
		Sink:    ch,
		source:  ch,
		decoder: decoder,
		pool:    sync.Pool{New: New},
	}
}

type BaseProvider struct {
	Sink    chan<- core.Ammo
	decoder Decoder
	source  <-chan core.Ammo
	pool    sync.Pool
}

func (ap *BaseProvider) Source() <-chan core.Ammo {
	return ap.source
}

func (ap *BaseProvider) Release(a core.Ammo) {
	ap.pool.Put(a)
}

func (ap *BaseProvider) Decode(src []byte) (*Ammo, error) {
	a := ap.pool.Get().(*Ammo)
	return ap.decoder.Decode(src, a)
}
