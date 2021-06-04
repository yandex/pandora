package engine

import (
	"testing"

	"github.com/yandex/pandora/lib/ginkgoutil"
	"github.com/yandex/pandora/lib/monitoring"
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
