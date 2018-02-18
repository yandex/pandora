// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package provider

import (
	"io"

	"github.com/json-iterator/go"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coreutil"
	"github.com/yandex/pandora/lib/ioutil2"
)

func NewJSONProvider(newAmmo func() core.Ammo, conf JSONProviderConfig) core.Provider {
	var newDecoder NewAmmoDecoder = func(deps core.ProviderDeps, source io.Reader) (AmmoDecoder, error) {
		return NewJSONAmmoDecoder(source, conf.Buffer.BufferSizeOrDefault()), nil
	}
	return NewDecodeProvider(newAmmo, newDecoder, conf.Decode)
}

type JSONProviderConfig struct {
	Decode DecodeProviderConfig      `config:",squash"`
	Buffer coreutil.BufferSizeConfig `config:",squash"`
}

func DefaultJSONProviderConfig() JSONProviderConfig {
	return JSONProviderConfig{Decode: DefaultDecodeProviderConfig()}
}

func NewJSONAmmoDecoder(r io.Reader, buffSize int) AmmoDecoder {
	var readError error
	// HACK(skipor): jsoniter.Iterator don't handle read errors well, but jsoniter.Decoder don't allow to set buffer size.
	var errTrackingReader ioutil2.ReaderFunc = func(p []byte) (n int, err error) {
		n, err = r.Read(p)
		if n > 0 {
			// Need to suppress error, to distinguish parse error in last chunk and read error.
			return n, nil
		}
		if err != nil {
			readError = err
		}
		return n, err
	}
	return &JSONAmmoDecoder{
		iter:         jsoniter.Parse(jsoniter.ConfigFastest, errTrackingReader, buffSize),
		readErrorPtr: &readError,
	}
}

type JSONAmmoDecoder struct {
	iter         *jsoniter.Iterator
	readErrorPtr *error
}

func (d *JSONAmmoDecoder) Decode(ammo core.Ammo) error {
	coreutil.ResetReusedAmmo(ammo)
	d.iter.ReadVal(ammo)
	if d.iter.Error != nil {
		if *d.readErrorPtr != nil {
			return *d.readErrorPtr
		}
		return d.iter.Error
	}
	return nil
}
