package scenario

import (
	"github.com/spf13/afero"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
)

func Import(fs afero.Fs) {
	register.Provider("http/scenario", func(cfg Config) (core.Provider, error) {
		return NewProvider(fs, cfg)
	})
}
