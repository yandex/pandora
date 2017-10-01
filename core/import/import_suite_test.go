package core

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/coretest"
	"github.com/yandex/pandora/lib/testutil"
)

func TestImport(t *testing.T) {
	Import(afero.NewOsFs())
	testutil.RunSuite(t, "Import Suite")
}

var _ = Describe("plugin decode", func() {

	Context("composite schedule", func() {
		input := func() map[string]interface{} {
			return map[string]interface{}{
				"schedule": []map[string]interface{}{
					{"type": "once", "times": 1},
					{"type": "const", "ops": 1, "duration": "1s"},
				},
			}
		}

		It("plugin", func() {
			var conf struct {
				Schedule core.Schedule
			}
			err := config.Decode(input(), &conf)
			Expect(err).NotTo(HaveOccurred())
			coretest.ExpectScheduleNexts(conf.Schedule, 0, time.Second, time.Second)
		})

		It("plugin factory", func() {
			var conf struct {
				Schedule func() (core.Schedule, error)
			}
			err := config.Decode(input(), &conf)
			Expect(err).NotTo(HaveOccurred())
			sched, err := conf.Schedule()
			Expect(err).NotTo(HaveOccurred())
			coretest.ExpectScheduleNexts(sched, 0, time.Second, time.Second)
		})
	})

})
