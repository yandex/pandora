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

	s.log = setupLogsCapture()
	s.metrics = newEngineMetrics()
}

func (s *PandoraSuite) Test_Http() {
	var requetsCount atomic.Int64 // Request served by test server.
	requetsCount.Store(0)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			requetsCount.Inc()
			rw.WriteHeader(http.StatusOK)
		}))
	defer srv.Close()

	conf := s.parseConfigFile("testdata/http/http.yaml", srv.Listener.Addr().String())
	s.Require().Equal(1, len(conf.Engine.Pools))
	aggr := &aggregator{}
	conf.Engine.Pools[0].Aggregator = aggr
	pandora := engine.New(s.log, s.metrics, conf.Engine)

	srv.Start()
	err := pandora.Run(context.Background())
	s.Assert().Equal(int64(4), requetsCount.Load())
	s.Require().NoError(err)
	s.Require().Equal(4, len(aggr.samples))
}

func (s *PandoraSuite) Test_Https() {
	var requetsCount atomic.Int64 // Request served by test server.
	requetsCount.Store(0)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			requetsCount.Inc()
			rw.WriteHeader(http.StatusOK)
		}))
	defer srv.Close()

	conf := s.parseConfigFile("testdata/http/https.yaml", srv.Listener.Addr().String())
	s.Require().Equal(1, len(conf.Engine.Pools))
	aggr := &aggregator{}
	conf.Engine.Pools[0].Aggregator = aggr
	pandora := engine.New(s.log, s.metrics, conf.Engine)

	srv.StartTLS()
	err := pandora.Run(context.Background())
	s.Assert().Equal(int64(4), requetsCount.Load())
	s.Require().NoError(err)
	s.Require().Equal(4, len(aggr.samples))
}

func (s *PandoraSuite) Test_Http2() {
	var requetsCount atomic.Int64 // Request served by test server.
	requetsCount.Store(0)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			requetsCount.Inc()
			rw.WriteHeader(http.StatusOK)
		}))
	defer srv.Close()

	conf := s.parseConfigFile("testdata/http/http2.yaml", srv.Listener.Addr().String())
	s.Require().Equal(1, len(conf.Engine.Pools))
	aggr := &aggregator{}
	conf.Engine.Pools[0].Aggregator = aggr
	pandora := engine.New(s.log, s.metrics, conf.Engine)

	_ = http2.ConfigureServer(srv.Config, nil)
	srv.TLS = srv.Config.TLSConfig
	srv.StartTLS()

	err := pandora.Run(context.Background())
	s.Assert().Equal(int64(4), requetsCount.Load())
	s.Require().NoError(err)
	s.Require().Equal(4, len(aggr.samples))
}

func (s *PandoraSuite) Test_Http2_UnsupportTarget() {
	var requetsCount atomic.Int64 // Request served by test server.
	requetsCount.Store(0)
	srv := httptest.NewUnstartedServer(http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			requetsCount.Inc()
			rw.WriteHeader(http.StatusOK)
		}))
	defer srv.Close()

	conf := s.parseConfigFile("testdata/http/http2.yaml", srv.Listener.Addr().String())
	s.Require().Equal(1, len(conf.Engine.Pools))
	aggr := &aggregator{}
	conf.Engine.Pools[0].Aggregator = aggr
	pandora := engine.New(s.log, s.metrics, conf.Engine)

	//_ = http2.ConfigureServer(srv.Config, nil)
	//srv.TLS = srv.Config.TLSConfig
	srv.StartTLS()

	err := pandora.Run(context.Background())
	s.Assert().Equal(int64(0), requetsCount.Load())
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "shoot panic: Non HTTP/2 connection established. Seems that target doesn't support HTTP/2.")
}

func (s *PandoraSuite) parseConfigFile(filename string, serverAddr string) *cli.CliConfig {
	f, err := os.ReadFile(filename)
	s.Require().NoError(err)
	tmpl, err := template.New("x").Parse(string(f))
	s.Require().NoError(err)
	b := &bytes.Buffer{}
	err = tmpl.Execute(b, map[string]string{"target": serverAddr})
	s.Require().NoError(err)
	mapCfg := map[string]any{}
	err = yaml.Unmarshal(b.Bytes(), &mapCfg)
	s.Require().NoError(err)

	conf := cli.DefaultConfig()
	err = config.DecodeAndValidate(mapCfg, conf)
	s.Require().NoError(err)

	return conf
}

func setupLogsCapture() *zap.Logger {
	c, _ := observer.New(zap.InfoLevel)
	return zap.New(c)
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
	samples []core.Sample
}

func (a *aggregator) Run(ctx context.Context, deps core.AggregatorDeps) error {
	return nil
}

func (a *aggregator) Report(s core.Sample) {
	a.samples = append(a.samples, s)
}
