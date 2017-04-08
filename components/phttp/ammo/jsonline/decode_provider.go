// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package jsonline

import (
	"sync"

	"github.com/yandex/pandora/core"
)

// FIXME(skipor): make type safe
type Decoder interface {
	Decode([]byte, core.Ammo) (core.Ammo, error)
}

func NewDecodeProvider(bufSize int, decoder Decoder, New func() interface{}) *DecodeProvider {
	ch := make(chan core.Ammo, bufSize)
	return &DecodeProvider{
		Sink:    ch,
		source:  ch,
		decoder: decoder,
		pool:    sync.Pool{New: New},
	}
}

type DecodeProvider struct {
	Sink    chan<- core.Ammo
	decoder Decoder
	source  <-chan core.Ammo
	pool    sync.Pool
}

func (ap *DecodeProvider) Source() <-chan core.Ammo {
	return ap.source
}

func (ap *DecodeProvider) Release(a core.Ammo) {
	ap.pool.Put(a)
}

func (ap *DecodeProvider) Decode(src []byte) (core.Ammo, error) {
	a := ap.pool.Get()
	return ap.decoder.Decode(src, a)
}
