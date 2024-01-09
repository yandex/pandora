package templater

import (
	gun "github.com/yandex/pandora/components/guns/http_scenario"
)

type Templater interface {
	Apply(request *gun.RequestParts, variables map[string]any, scenarioName, stepName string) error
}
