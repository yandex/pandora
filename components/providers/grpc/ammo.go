// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package ammo

type Ammo struct {
	Tag       string                 `json:"tag"`
	Call      string                 `json:"call"`
	Metadata  map[string]string      `json:"metadata"`
	Payload   map[string]interface{} `json:"payload"`
	id        uint64
	isInvalid bool
}

func (a *Ammo) Reset(tag string, call string, metadata map[string]string, payload map[string]interface{}) {
	*a = Ammo{tag, call, metadata, payload, 0, false}
}

func (a *Ammo) SetID(id uint64) {
	a.id = id
}

func (a *Ammo) ID() uint64 {
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
