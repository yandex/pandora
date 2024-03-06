package acceptance

import (
	"context"
	"net/http"
	"net/http/httptest"
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
)

func TestConnectGunSuite(t *testing.T) {
	suite.Run(t, new(ConnectGunSuite))
}

type ConnectGunSuite struct {
	suite.Suite
	fs      afero.Fs
	log     *zap.Logger
	metrics engine.Metrics
}

func (s *ConnectGunSuite) SetupSuite() {
	s.fs = afero.NewOsFs()
	testOnce.Do(func() {
		coreimport.Import(s.fs)
		phttpimport.Import(s.fs)
		grpc.Import(s.fs)
	})

	s.log = testutil.NewNullLogger()
	s.metrics = newEngineMetrics("connect_suite")
}

func (s *ConnectGunSuite) Test_Connect() {
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
			filecfg: "testdata/connect/connect.yaml",
			isTLS:   false,
			wantCnt: 4,
		},
		{
			name:    "http-check-limits",
			filecfg: "testdata/connect/connect-check-limit.yaml",
			isTLS:   false,
			wantCnt: 8,
		},
		{
			name:    "http-check-passes",
			filecfg: "testdata/connect/connect-check-passes.yaml",
			isTLS:   false,
			wantCnt: 15,
		},
		// TODO: first record does not look like a TLS handshake. Check https://go.dev/src/crypto/tls/conn.go
		//{
		//	name:    "connect-ssl",
		//	filecfg: "testdata/connect/connect-ssl.yaml",
		//	isTLS:   true,
		//	wantCnt: 4,
		//},
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
			s.Require().Equal(int64(tt.wantCnt), int64(len(aggr.samples)))
			s.Assert().GreaterOrEqual(requetsCount.Load(), int64(len(aggr.samples))) // requetsCount more than shoots
		})
	}
}
