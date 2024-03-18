package coreutil

import (
	"github.com/c2h5oh/datasize"
)

const DefaultBufferSize = 512 * 1024
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
