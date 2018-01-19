package engine

import (
	"testing"

	"github.com/yandex/pandora/lib/monitoring"
	"github.com/yandex/pandora/lib/testutil"
)

func TestEngine(t *testing.T) {
	testutil.RunSuite(t, "Engine Suite")
}

func newTestMetrics() Metrics {
	return Metrics{
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
	}
}
