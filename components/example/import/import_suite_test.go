package example

import (
	"testing"

	"a.yandex-team.ru/load/projects/pandora/lib/ginkgoutil"
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
