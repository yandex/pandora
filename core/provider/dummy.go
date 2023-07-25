package provider

import (
	"context"

	"github.com/yandex/pandora/core"
)

type Dummy struct {
}

func (d Dummy) Run(context.Context, core.ProviderDeps) error {
	return nil
}

func (d Dummy) Acquire() (ammo core.Ammo, ok bool) {
	return nil, true
}

func (d Dummy) Release(core.Ammo) {}

var _ core.Provider = (*Dummy)(nil)
