package httphttp2

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"text/template"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/yandex/pandora/cli"
	grpc "github.com/yandex/pandora/components/grpc/import"
	phttpimport "github.com/yandex/pandora/components/phttp/import"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/engine"
	coreimport "github.com/yandex/pandora/core/import"
	"github.com/yandex/pandora/lib/monitoring"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"golang.org/x/net/http2"
	"gopkg.in/yaml.v2"
)

var testOnce = &sync.Once{}

func TestGunSuite(t *testing.T) {
	suite.Run(t, new(PandoraSuite))
}

type PandoraSuite struct {
	suite.Suite
	fs      afero.Fs
	log     *zap.Logger
	metrics engine.Metrics
}

func (s *PandoraSuite) SetupSuite() {
	s.fs = afero.NewOsFs()
	coreimport.Import(s.fs)
	phttpimport.Import(s.fs)
	grpc.Import(s.fs)

	s.log = newNullLogger()
	// s.log = newLogger()
	s.metrics = newEngineMetrics()
}

func (s *PandoraSuite) Test_Http_Check_Passes() {
	tests := []struct {
		name           string
		filecfg        string
		isTLS          bool
		preStartSrv    func(srv *httptest.Server)
		wantErrContain string
		wantCnt        int
	}{
		{
			name:    "http",
			filecfg: "testdata/http/http.yaml",
			isTLS:   false,
			wantCnt: 4,
		},
		{
			name:    "https",
			filecfg: "testdata/http/https.yaml",
			isTLS:   true,
			wantCnt: 4,
		},
		{
			name:    "http2",
			filecfg: "testdata/http/http2.yaml",
			isTLS:   true,
			preStartSrv: func(srv *httptest.Server) {
				_ = http2.ConfigureServer(srv.Config, nil)
				srv.TLS = srv.Config.TLSConfig
			},
			wantCnt: 4,
		},
		{
			name:    "http2 unsapported",
			filecfg: "testdata/http/http2.yaml",
			isTLS:   true,
			preStartSrv: func(srv *httptest.Server) {
				//_ = http2.ConfigureServer(srv.Config, nil)
				//srv.TLS = srv.Config.TLSConfig
			},
			wantErrContain: "shoot panic: Non HTTP/2 connection established. Seems that target doesn't support HTTP/2.",
		},
		{
			name:    "http-check-limits",
			filecfg: "testdata/http/http-check-limit.yaml",
			isTLS:   false,
			wantCnt: 8,
		},
		{
			name:    "http-check-passes",
			filecfg: "testdata/http/http-check-passes.yaml",
			isTLS:   false,
			wantCnt: 15,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			var requetsCount atomic.Int64 // Request served by test server.
			requetsCount.Store(0)
			srv := httptest.NewUnstartedServer(http.HandlerFunc(
				func(rw http.ResponseWriter, req *http.Request) {
					requetsCount.Inc()
					rw.WriteHeader(http.StatusOK)
				}))
			defer srv.Close()

			conf := parseConfigFile(s.T(), tt.filecfg, srv.Listener.Addr().String())
			s.Require().Equal(1, len(conf.Engine.Pools))
			aggr := &aggregator{}
			conf.Engine.Pools[0].Aggregator = aggr
			pandora := engine.New(s.log, s.metrics, conf.Engine)

			if tt.preStartSrv != nil {
				tt.preStartSrv(srv)
			}
			if tt.isTLS {
				srv.StartTLS()
			} else {
				srv.Start()
			}
			err := pandora.Run(context.Background())
			if tt.wantErrContain != "" {
				s.Assert().Equal(int64(0), requetsCount.Load())
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tt.wantErrContain)
				return
			}
			s.Require().NoError(err)
			s.Assert().Equal(int64(tt.wantCnt), requetsCount.Load())
			s.Require().Equal(tt.wantCnt, len(aggr.samples))
		})
	}
}

func parseConfigFile(t *testing.T, filename string, serverAddr string) *cli.CliConfig {
	mapCfg := unmarshalConfigFile(t, filename, serverAddr)
	conf := decodeConfig(t, mapCfg)
	return conf
}

func decodeConfig(t *testing.T, mapCfg map[string]any) *cli.CliConfig {
	conf := cli.DefaultConfig()
	err := config.DecodeAndValidate(mapCfg, conf)
	require.NoError(t, err)
	return conf
}

func unmarshalConfigFile(t *testing.T, filename string, serverAddr string) map[string]any {
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

func newNullLogger() *zap.Logger {
	c, _ := observer.New(zap.InfoLevel)
	return zap.New(c)
}

func newLogger() *zap.Logger {
	zapConf := zap.NewDevelopmentConfig()
	zapConf.Level.SetLevel(zapcore.DebugLevel)
	log, err := zapConf.Build(zap.AddCaller())
	if err != nil {
		zap.L().Fatal("Logger build failed", zap.Error(err))
	}
	return log
}

func newEngineMetrics() engine.Metrics {
	return engine.Metrics{
		Request:        monitoring.NewCounter("engine_Requests"),
		Response:       monitoring.NewCounter("engine_Responses"),
		InstanceStart:  monitoring.NewCounter("engine_UsersStarted"),
		InstanceFinish: monitoring.NewCounter("engine_UsersFinished"),
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
