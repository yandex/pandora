package httpscenario

import "github.com/yandex/pandora/core/register"

type VariableSource interface {
	GetName() string
	GetVariables() any
	Init() error
}

func RegisterVariableSource(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *VariableSource
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}

type SourceStorage struct {
	sources map[string]any
}

func (s *SourceStorage) AddSource(name string, variables any) {
	s.sources[name] = variables
}

func (s *SourceStorage) Variables() map[string]any {
	return s.sources
}
