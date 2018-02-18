// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coreutil

import (
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/magiconair/properties/assert"
)

func TestBufferSizeConfig_BufferSizeOrDefault(t *testing.T) {
	get := func(s datasize.ByteSize) int {
		return BufferSizeConfig{s}.BufferSizeOrDefault()
	}
	assert.Equal(t, DefaultBufferSize, get(0))
	assert.Equal(t, MinimalBufferSize, get(1))
	const big = DefaultBufferSize * DefaultBufferSize
	assert.Equal(t, big, get(big))
}
