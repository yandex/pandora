package httpscenario

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	phttp "github.com/yandex/pandora/components/guns/http"
	httpscenario "github.com/yandex/pandora/components/guns/http_scenario"
	ammo "github.com/yandex/pandora/components/providers/scenario"
	httpammo "github.com/yandex/pandora/components/providers/scenario/http"
	_import "github.com/yandex/pandora/components/providers/scenario/import"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/plugin/pluginconfig"
	"github.com/yandex/pandora/examples/http/server"
	"go.uber.org/zap"
)

var testOnce = &sync.Once{}

func TestGunSuite(t *testing.T) {
	suite.Run(t, new(GunSuite))
}

type GunSuite struct {
	suite.Suite
	server *server.Server
	addr   string
	fs     afero.Fs
}

func (s *GunSuite) SetupSuite() {
	s.fs = afero.NewOsFs()
	httpscenario.Import(s.fs)
	_import.Import(s.fs)
	testOnce.Do(func() {
		pluginconfig.AddHooks()
	})

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	port := os.Getenv("PORT") // TODO: how to set free port in CI?
	if port == "" {
		port = "8886"
	}

	s.addr = "localhost:" + port
	s.server = server.NewServer(s.addr, logger, time.Now().UnixNano())
	s.server.ServeAsync()

	go func() {
		err := <-s.server.Err()
		s.NoError(err)
	}()
}

func (s *GunSuite) TearDownSuite() {
	err := s.server.Shutdown(context.Background())
	s.NoError(err)
}

func (s *GunSuite) SetupTest() {
	s.server.Stats().Reset()
}

func (s *GunSuite) Test_SuccessScenario() {
	ctx := context.Background()
	log := zap.NewNop()
	g := httpscenario.NewHTTPGun(phttp.HTTPGunConfig{
		Gun: phttp.GunConfig{
			Target: s.addr,
		},
		Client: phttp.ClientConfig{},
	}, log, s.addr)

	gunDeps := core.GunDeps{Ctx: ctx, Log: log, PoolID: "pool_id", InstanceID: 1}
	aggr := &Aggregator{}
	err := g.Bind(aggr, gunDeps)
	s.NoError(err)

	pr, err := httpammo.NewProvider(s.fs, ammo.ProviderConfig{File: "testdata/http_payload.hcl"})
	require.NoError(s.T(), err)
	go func() {
		_ = pr.Run(ctx, core.ProviderDeps{Log: log, PoolID: "pool_id"})
	}()

	for i := 0; i < 3; i++ {
		am, ok := pr.Acquire()
		s.True(ok)
		scenario, ok := am.(*httpscenario.Scenario)
		s.True(ok)
		g.Shoot(scenario)
	}

	stats := s.server.Stats()
	s.Equal(map[int64]uint64{1: 1, 2: 1, 3: 1}, stats.Auth200)
	s.Equal(map[int64]uint64{1: 3, 2: 3, 3: 3}, stats.Order200)
}

type Aggregator struct {
	samples []*netsample.Sample
}

func (a *Aggregator) Run(ctx context.Context, deps core.AggregatorDeps) error {
	return nil
}

func (a *Aggregator) Report(s *netsample.Sample) {
	a.samples = append(a.samples, s)
}
