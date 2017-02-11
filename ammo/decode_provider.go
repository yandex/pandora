// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package ammo

import "sync"

type Decoder interface {
	Decode([]byte, Ammo) (Ammo, error)
}

func NewDecodeProvider(bufSize int, decoder Decoder, New func() interface{}) *DecodeProvider {
	ch := make(chan Ammo, bufSize)
	return &DecodeProvider{
		Sink:    ch,
		source:  ch,
		decoder: decoder,
		pool:    sync.Pool{New: New},
	}
}

type DecodeProvider struct {
	Sink    chan<- Ammo
	decoder Decoder
	source  <-chan Ammo
	pool    sync.Pool
}

func (ap *DecodeProvider) Source() <-chan Ammo {
	return ap.source
}

func (ap *DecodeProvider) Release(a Ammo) {
	ap.pool.Put(a)
}

func (ap *DecodeProvider) Decode(src []byte) (Ammo, error) {
	a := ap.pool.Get()
	return ap.decoder.Decode(src, a)
}
