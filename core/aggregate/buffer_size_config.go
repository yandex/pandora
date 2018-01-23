// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregate

import (
	"github.com/c2h5oh/datasize"
)

const DefaultBufferSize = 256 * 1024
const MinimalBufferSize = 4 * 1024

// BufferSizeConfig SHOULD be used to configure buffer size.
// That makes buffer size configuration consistent among all Aggregators.
type BufferSizeConfig struct {
	BufferSize datasize.ByteSize `config:"buffer-size"`
}

func (conf BufferSizeConfig) BufferSizeOrDefault() int {
	bufSize := int(conf.BufferSize)
	if bufSize == 0 {
		return DefaultBufferSize
	}
	if bufSize <= MinimalBufferSize {
		return MinimalBufferSize
	}
	return bufSize
}
