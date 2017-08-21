package engine

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"

	"github.com/yandex/pandora/lib/monitoring"
)

func TestEngine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Engine Suite")
}

func newTestMetrics() Metrics {
	return Metrics{
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
		&monitoring.Counter{},
	}
}
