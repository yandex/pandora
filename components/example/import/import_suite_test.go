package example

import (
	"testing"

	"github.com/yandex/pandora/lib/ginkgoutil"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestImport(t *testing.T) {
	ginkgoutil.RunSuite(t, "Import Suite")
}

var _ = Describe("import", func() {
	It("not panics", func() {
		Expect(Import).NotTo(Panic())
	})
})
