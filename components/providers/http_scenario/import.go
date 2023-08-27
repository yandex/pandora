package httpscenario

import (
	"sync"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/providers/http_scenario/postprocessor"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
)

var once = &sync.Once{}

func Import(fs afero.Fs) {
	once.Do(func() {
		register.Provider("http/scenario", func(cfg Config) (core.Provider, error) {
			return NewProvider(fs, cfg)
		})
	})
}

func RegisterPostprocessor(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *postprocessor.Postprocessor
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}
