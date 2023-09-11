package httpscenario

//go:generate go run github.com/vektra/mockery/v2@v2.22.1 --inpackage --name=Templater --filename=mock_templater_test.go

type Templater interface {
	Apply(request *RequestParts, variables map[string]any, scenarioName, stepName string) error
}
