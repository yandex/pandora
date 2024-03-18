package acceptance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/suite"
	grpc "github.com/yandex/pandora/components/grpc/import"
	phttpimport "github.com/yandex/pandora/components/phttp/import"
	"github.com/yandex/pandora/core/engine"
	coreimport "github.com/yandex/pandora/core/import"
	"github.com/yandex/pandora/lib/testutil"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
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
	testOnce.Do(func() {
		coreimport.Import(s.fs)
		phttpimport.Import(s.fs)
		grpc.Import(s.fs)
	})

	s.log = testutil.NewNullLogger()
	s.metrics = newEngineMetrics("http_suite")
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
		{
			name:    "http2-pool-size",
			filecfg: "testdata/http/http2-pool-size.yaml",
			isTLS:   true,
			preStartSrv: func(srv *httptest.Server) {
				_ = http2.ConfigureServer(srv.Config, nil)
				srv.TLS = srv.Config.TLSConfig
			},
			wantCnt: 8,
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
