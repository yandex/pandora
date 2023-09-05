package httpscenario

import httpscenario "github.com/yandex/pandora/components/guns/http_scenario"

type Templater interface {
	Apply(request *httpscenario.RequestParts, variables map[string]any, scenarioName, stepName string) error
}
