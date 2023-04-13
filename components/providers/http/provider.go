// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.

package http

import (
	"bytes"
	"io"
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
	var (
		readSeeker io.ReadSeeker
		closer     io.Closer
		err        error
	)
	if len(conf.Uris) > 0 {
		readSeeker, closer, err = uriReadSeekCloser(conf)
	} else {
		readSeeker, closer, err = fileReadSeekCloser(fs, conf.File)
	}
	if err != nil {
		return nil, xerrors.Errorf("cant create ReadSeekCloser: %w", err)
	}
	decoder, err := decoders.NewDecoder(conf, readSeeker)
	if err != nil {
		return nil, xerrors.Errorf("decoder init error: %w", err)
	}
	return &provider.Provider{
		ProviderBase: base.ProviderBase{
			FS: fs,
		},
		Config:   conf,
		Decoder:  decoder,
		Close:    closer.Close,
		AmmoPool: sync.Pool{New: func() interface{} { return new(base.Ammo[http.Request]) }},
		Sink:     make(chan *base.Ammo[http.Request]),
	}, nil
}

func fileReadSeekCloser(fs afero.Fs, path string) (io.ReadSeeker, io.Closer, error) {
	if path == "" {
		return nil, nil, xerrors.Errorf("one should specify either 'file' or 'uris'")
	}
	file, err := fs.Open(path)
	if err != nil {
		return nil, nil, xerrors.Errorf("open file error: %w", err)
	}
	return file, file, nil
}

func uriReadSeekCloser(conf config.Config) (io.ReadSeeker, io.Closer, error) {
	if conf.Decoder != config.DecoderURI {
		return nil, nil, xerrors.Errorf("'uris' expect setted only for 'uri' decoder, but faced with '%s'", conf.Decoder)
	}
	if conf.File != "" {
		return nil, nil, xerrors.Errorf("one should specify either 'file' or 'uris', but not both of them")
	}
	reader := bytes.NewReader([]byte(strings.Join(conf.Uris, "\n")))
	readSeeker := io.ReadSeeker(reader)
	return readSeeker, nil, nil
}
