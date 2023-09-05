package httpscenario

type VariableSource interface {
	GetName() string
	GetVariables() any
	Init() error
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
