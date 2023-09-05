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

		RegisterVariableSource("file/csv", func(cfg VariableSourceCsv) (VariableSource, error) {
			return NewVSCSV(cfg, fs)
		})

		RegisterVariableSource("file/json", func(cfg VariableSourceJSON) (VariableSource, error) {
			return NewVSJson(cfg, fs)
		})

		RegisterVariableSource("variables", func(cfg VariableSourceVariables) VariableSource {
			return &cfg
		})

		RegisterPostprocessor("var/jsonpath", postprocessor.NewVarJsonpathPostprocessor)
		RegisterPostprocessor("var/xpath", postprocessor.NewVarXpathPostprocessor)
		RegisterPostprocessor("var/header", postprocessor.NewVarHeaderPostprocessor)
		RegisterPostprocessor("assert/response", postprocessor.NewAssertResponsePostprocessor)

		RegisterTemplater("text", NewTextTemplater)
		RegisterTemplater("html", NewHTMLTemplater)
	})
}

func RegisterPostprocessor(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *postprocessor.Postprocessor
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}

func RegisterVariableSource(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *VariableSource
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}

func RegisterTemplater(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *Templater
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}
