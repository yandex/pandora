package coreimport

import (
	"context"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/coretest"
	"github.com/yandex/pandora/core/plugin"
	"github.com/yandex/pandora/lib/ginkgoutil"
	"github.com/yandex/pandora/lib/testutil"
)

func TestImport(t *testing.T) {
	defer resetGlobals()
	Import(afero.NewOsFs())
	ginkgoutil.RunSuite(t, "Import Suite")
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
			coretest.ExpectScheduleNexts(conf.Schedule, 0, 0, time.Second)
		})

		It("plugin factory", func() {
			var conf struct {
				Schedule func() (core.Schedule, error)
			}
			err := config.Decode(input(), &conf)
			Expect(err).NotTo(HaveOccurred())
			sched, err := conf.Schedule()
			Expect(err).NotTo(HaveOccurred())
			coretest.ExpectScheduleNexts(sched, 0, 0, time.Second)
		})
	})
})

func TestSink(t *testing.T) {
	defer resetGlobals()
	fs := afero.NewMemMapFs()
	const filename = "/xxx"
	Import(fs)

	tests := []struct {
		name  string
		input map[string]interface{}
	}{
		{"hooked", testConfig(
			"stdout", "stdout",
			"stderr", "stderr",
			"file", filename,
		)},
		{"explicit", testConfig(
			"stdout", testConfig("type", "stdout"),
			"stderr", testConfig("type", "stderr"),
			"file", testConfig(
				"type", "file",
				"path", filename,
			),
		)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var conf struct {
				Stdout func() core.DataSink
				Stderr func() core.DataSink
				File   core.DataSink
			}
			err := config.Decode(test.input, &conf)
			require.NoError(t, err)
			coretest.AssertSinkEqualStdStream(t, &os.Stdout, conf.Stdout)
			coretest.AssertSinkEqualStdStream(t, &os.Stderr, conf.Stderr)
			coretest.AssertSinkEqualFile(t, fs, filename, conf.File)
		})
	}
}

func TestProviderJSONLine(t *testing.T) {
	testutil.ReplaceGlobalLogger()
	defer resetGlobals()
	fs := afero.NewMemMapFs()
	const filename = "/xxx"
	Import(fs)
	input := testConfig(
		"aggregator", testConfig(
			"type", "jsonlines",
			"sink", filename,
		),
	)

	var conf struct {
		Aggregator core.Aggregator
	}
	err := config.Decode(input, &conf)
	require.NoError(t, err)

	conf.Aggregator.Report([]int{0, 1, 2})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = conf.Aggregator.Run(ctx, core.AggregatorDeps{Log: zap.L()})
	require.NoError(t, err)

	testutil.AssertFileEqual(t, fs, filename, "[0,1,2]\n")
}

// TODO(skipor): test datasources

func testConfig(keyValuePairs ...interface{}) map[string]interface{} {
	if len(keyValuePairs)%2 != 0 {
		panic("invalid len")
	}
	result := map[string]interface{}{}
	for i := 0; i < len(keyValuePairs); i += 2 {
		key := keyValuePairs[i].(string)
		value := keyValuePairs[i+1]
		result[key] = value
	}
	return result
}

func resetGlobals() {
	plugin.SetDefaultRegistry(plugin.NewRegistry())
	config.SetHooks(config.DefaultHooks())
	testutil.ReplaceGlobalLogger()
}
