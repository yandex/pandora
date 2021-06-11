package engine

import (
	"testing"

	"a.yandex-team.ru/load/projects/pandora/lib/ginkgoutil"
	"a.yandex-team.ru/load/projects/pandora/lib/monitoring"
)

func TestEngine(t *testing.T) {
	ginkgoutil.RunSuite(t, "Engine Suite")
}

func newTestMetrics() Metrics {
	return Metrics{
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
	}
}
