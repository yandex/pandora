package httpscenario

type Templater interface {
	Apply(request *requestParts, variables map[string]any, scenarioName, stepName string) error
}
