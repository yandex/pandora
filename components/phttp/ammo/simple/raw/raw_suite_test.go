package raw

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRaw(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Raw Suite")
}
