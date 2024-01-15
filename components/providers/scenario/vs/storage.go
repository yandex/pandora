package vs

func NewVariableStorage() *SourceStorage {
	return &SourceStorage{
		sources: make(map[string]any),
	}
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
