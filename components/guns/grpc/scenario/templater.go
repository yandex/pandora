package scenario

type Templater interface {
	Apply(payload []byte, metadata map[string]string, variables map[string]any, scenarioName, stepName string) ([]byte, error)
}
