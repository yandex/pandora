package scenario

import (
	"sync"

	"github.com/spf13/afero"
	grpcgun "github.com/yandex/pandora/components/guns/grpc/scenario"
	gun "github.com/yandex/pandora/components/guns/http_scenario"
	"github.com/yandex/pandora/components/providers/scenario"
	"github.com/yandex/pandora/components/providers/scenario/grpc"
	grpcpostprocessor "github.com/yandex/pandora/components/providers/scenario/grpc/postprocessor"
	grpcpreprocessor "github.com/yandex/pandora/components/providers/scenario/grpc/preprocessor"
	"github.com/yandex/pandora/components/providers/scenario/http"
	"github.com/yandex/pandora/components/providers/scenario/http/postprocessor"
	"github.com/yandex/pandora/components/providers/scenario/http/templater"
	"github.com/yandex/pandora/components/providers/scenario/vs"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
)

var once = &sync.Once{}

func Import(fs afero.Fs) {
	once.Do(func() {
		register.Provider("http/scenario", func(cfg scenario.ProviderConfig) (core.Provider, error) {
			return http.NewProvider(fs, cfg)
		})
		register.Provider("grpc/scenario", func(cfg scenario.ProviderConfig) (core.Provider, error) {
			return grpc.NewProvider(fs, cfg)
		})

		RegisterVariableSource("file/csv", func(cfg vs.VariableSourceCsv) (vs.VariableSource, error) {
			return vs.NewVSCSV(cfg, fs)
		})

		RegisterVariableSource("file/json", func(cfg vs.VariableSourceJSON) (vs.VariableSource, error) {
			return vs.NewVSJson(cfg, fs)
		})

		RegisterVariableSource("variables", func(cfg vs.VariableSourceVariables) vs.VariableSource {
			return &cfg
		})

		RegisterPostprocessor("var/jsonpath", NewVarJsonpathPostprocessor)
		RegisterPostprocessor("var/xpath", NewVarXpathPostprocessor)
		RegisterPostprocessor("var/header", NewVarHeaderPostprocessor)
		RegisterPostprocessor("assert/response", NewAssertResponsePostprocessor)
		RegisterPostprocessor("assert/json-schema", NewAssertJsonSchemaPostprocessor)

		RegisterTemplater("text", func() gun.Templater {
			return templater.NewTextTemplater()
		})
		RegisterTemplater("html", func() gun.Templater {
			return templater.NewHTMLTemplater()
		})

		RegisterGRPCPostprocessor("assert/response", func(cfg grpcpostprocessor.AssertResponse) grpcgun.Postprocessor {
			return &cfg
		})
		RegisterGRPCPreprocessor("prepare", func(cfg grpcpreprocessor.PreprocessorConfig) grpcgun.Preprocessor {
			return &grpcpreprocessor.PreparePreprocessor{Mapping: cfg.Mapping}
		})
	})
}

func RegisterGRPCPreprocessor(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *grpcgun.Preprocessor
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}

func RegisterGRPCPostprocessor(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *grpcgun.Postprocessor
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}

func RegisterTemplater(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *gun.Templater
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}

func RegisterVariableSource(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *vs.VariableSource
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}

func RegisterPostprocessor(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *gun.Postprocessor
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}

func NewAssertResponsePostprocessor(cfg postprocessor.AssertResponse) (gun.Postprocessor, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func NewVarHeaderPostprocessor(cfg postprocessor.Config) gun.Postprocessor {
	return &postprocessor.VarHeaderPostprocessor{
		Mapping: cfg.Mapping,
	}
}

func NewVarJsonpathPostprocessor(cfg postprocessor.Config) gun.Postprocessor {
	return &postprocessor.VarJsonpathPostprocessor{
		Mapping: cfg.Mapping,
	}
}

func NewVarXpathPostprocessor(cfg postprocessor.Config) gun.Postprocessor {
	return &postprocessor.VarXpathPostprocessor{
		Mapping: cfg.Mapping,
	}
}

func NewAssertJsonSchemaPostprocessor(cfg postprocessor.AssertJsonSchema) (gun.Postprocessor, error) {
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
