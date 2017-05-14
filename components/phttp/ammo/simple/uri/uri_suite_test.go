package uri

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestUri(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Uri Suite")
}
