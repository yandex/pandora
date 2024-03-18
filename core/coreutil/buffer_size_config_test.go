package coreutil

import (
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/assert"
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
