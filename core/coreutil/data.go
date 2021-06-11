// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coreutil

import (
	"io"

	"a.yandex-team.ru/load/projects/pandora/core"
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
