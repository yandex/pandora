package netsample

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/yandex/pandora/lib/testutil"
)

func TestNetsample(t *testing.T) {
	RegisterFailHandler(Fail)
	testutil.ReplaceGlobalLogger()
	RunSpecs(t, "Netsample Suite")
}
