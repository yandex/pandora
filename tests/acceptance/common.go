package acceptance

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/engine"
	"github.com/yandex/pandora/lib/monitoring"
	"gopkg.in/yaml.v2"
)

func parseConfigFile(t *testing.T, filename string, serverAddr string) *cli.CliConfig {
	t.Helper()
	mapCfg := unmarshalConfigFile(t, filename, serverAddr)
	conf := decodeConfig(t, mapCfg)
	return conf
}

func decodeConfig(t *testing.T, mapCfg map[string]any) *cli.CliConfig {
	t.Helper()
	conf := cli.DefaultConfig()
	err := config.DecodeAndValidate(mapCfg, conf)
	require.NoError(t, err)
	return conf
}

func unmarshalConfigFile(t *testing.T, filename string, serverAddr string) map[string]any {
	t.Helper()
	f, err := os.ReadFile(filename)
	require.NoError(t, err)
	tmpl, err := template.New("x").Parse(string(f))
	require.NoError(t, err)
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, map[string]string{"target": serverAddr})
	require.NoError(t, err)
	mapCfg := map[string]any{}
	err = yaml.Unmarshal(b.Bytes(), &mapCfg)
	require.NoError(t, err)
	return mapCfg
}

func newEngineMetrics(prefix string) engine.Metrics {
	return engine.Metrics{
		Request:        monitoring.NewCounter(prefix + "_Requests"),
		Response:       monitoring.NewCounter(prefix + "_Responses"),
		InstanceStart:  monitoring.NewCounter(prefix + "_UsersStarted"),
		InstanceFinish: monitoring.NewCounter(prefix + "_UsersFinished"),
	}
}

type aggregator struct {
	mx      sync.Mutex
	samples []core.Sample
}

func (a *aggregator) Run(ctx context.Context, deps core.AggregatorDeps) error {
	return nil
}

func (a *aggregator) Report(s core.Sample) {
	a.mx.Lock()
	defer a.mx.Unlock()
	a.samples = append(a.samples, s)
}
