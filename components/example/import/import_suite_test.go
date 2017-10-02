package example

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/yandex/pandora/lib/testutil"
)

func TestImport(t *testing.T) {
	testutil.RunSuite(t, "Import Suite")
}

var _ = Describe("import", func() {
	It("not panics", func() {
		Expect(Import).NotTo(Panic())
	})
})
