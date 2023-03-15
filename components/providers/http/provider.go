// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package http

import (
	"net/http"
	"strings"
	"sync"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/providers/base"
	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders"
	"github.com/yandex/pandora/components/providers/http/provider"
	"github.com/yandex/pandora/core"
	"golang.org/x/xerrors"
)

func NewProvider(fs afero.Fs, conf config.Config) (core.Provider, error) {
	if !conf.Decoder.IsValid() {
		return nil, xerrors.Errorf("unknown decoder type faced")
	}
	if len(conf.Uris) > 0 {
		if conf.Decoder != config.DecoderURI {
			return nil, xerrors.Errorf("'uris' expect setted only for 'uri' decoder, but faced with '%s'", conf.Decoder)
		}
		if conf.File != "" {
			return nil, xerrors.Errorf("one should specify either 'file' or 'uris', but not both of them")
		}
		fs = afero.NewMemMapFs()
		conf.File = "ammo.uri"
		err := afero.WriteFile(fs, conf.File, []byte(strings.Join(conf.Uris, "\n")), 0444)
		if err != nil {
			return nil, xerrors.Errorf("uri based ammo file create error: %w", err)
		}
	}
	if conf.File == "" {
		return nil, xerrors.Errorf("one should specify either 'file' or 'uris'")
	}

	file, err := fs.Open(conf.File)
	if err != nil {
		return nil, xerrors.Errorf("open file error: %w", err)
	}
	decoder, err := decoders.NewDecoder(conf, file)
	if err != nil {
		return nil, xerrors.Errorf("decoder init error: %w", err)
	}
	p := &provider.Provider{
		ProviderBase: base.ProviderBase{
			FS: fs,
		},
		Config:   conf,
		Decoder:  decoder,
		Close:    file.Close,
		AmmoPool: sync.Pool{New: func() interface{} { return new(base.Ammo[http.Request]) }},
		Sink:     make(chan *base.Ammo[http.Request]),
	}
	return p, err
}
