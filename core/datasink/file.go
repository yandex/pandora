// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package datasink

import (
	"io"
	"os"

	"github.com/spf13/afero"

	"github.com/yandex/pandora/core"
)

type FileSinkConfig struct {
	Path string
}

func NewFileSink(fs afero.Fs, conf FileSinkConfig) core.DataSink {
	return &fileSink{afero.Afero{fs}, conf}
}

type fileSink struct {
	fs   afero.Afero
	conf FileSinkConfig
}

func (s *fileSink) OpenSink() (wc io.WriteCloser, err error) {
	return s.fs.OpenFile(s.conf.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func NewStdoutSink() core.DataSink {
	return hideCloseFileSink{os.Stdout}
}

func NewStderrSink() core.DataSink {
	return hideCloseFileSink{os.Stderr}
}

type hideCloseFileSink struct{ *os.File }

func (f hideCloseFileSink) OpenSink() (wc io.WriteCloser, err error) {
	return f, nil
}

func (f hideCloseFileSink) Close() error { return nil }
