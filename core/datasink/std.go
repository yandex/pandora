// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package datasink

import (
	"bytes"
	"io"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/ioutil2"
)

type Buffer struct {
	bytes.Buffer
	ioutil2.NopCloser
}

var _ core.DataSink = &Buffer{}

func (b *Buffer) OpenSink() (wc io.WriteCloser, err error) {
	return b, nil
}

func NewBuffer() *Buffer {
	return &Buffer{}
}
