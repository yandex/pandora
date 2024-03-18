package coreutil

import (
	"io"

	"github.com/yandex/pandora/core"
)

type DataSinkFunc func() (wc io.WriteCloser, err error)

func (f DataSinkFunc) OpenSink() (wc io.WriteCloser, err error) {
	return f()
}

var _ core.DataSink = DataSinkFunc(nil)

type DataSourceFunc func() (wc io.ReadCloser, err error)

func (f DataSourceFunc) OpenSource() (rc io.ReadCloser, err error) {
	return f()
}

var _ core.DataSource = DataSourceFunc(nil)
