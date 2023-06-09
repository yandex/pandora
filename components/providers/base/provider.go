package base

import (
	"sync/atomic"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/core"
)

type ProviderBase struct {
	Deps      core.ProviderDeps
	FS        afero.Fs
	idCounter atomic.Uint64
}

func (p *ProviderBase) NextID() uint64 {
	return p.idCounter.Add(1)
}
