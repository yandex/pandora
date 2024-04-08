package grpcscenario

import (
	"context"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	grpcscenario "github.com/yandex/pandora/components/guns/grpc/scenario"
	"github.com/yandex/pandora/components/providers/scenario"
	ammo "github.com/yandex/pandora/components/providers/scenario/grpc"
	"github.com/yandex/pandora/components/providers/scenario/grpc/postprocessor"
	_import "github.com/yandex/pandora/components/providers/scenario/import"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/plugin/pluginconfig"
	"github.com/yandex/pandora/core/warmup"
	"github.com/yandex/pandora/examples/grpc/server"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

var testOnce = &sync.Once{}

func TestGunSuite(t *testing.T) {
	suite.Run(t, new(GunSuite))
}

type GunSuite struct {
	suite.Suite
	grpcServer  *grpc.Server
	grpcAddress string
	fs          afero.Fs
	srv         *server.GRPCServer
}

func (s *GunSuite) SetupSuite() {
	s.fs = afero.NewOsFs()
	_import.Import(s.fs)
	testOnce.Do(func() {
		pluginconfig.AddHooks()
	})
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	port := os.Getenv("PORT") // TODO: how to set free port in CI?
	if port == "" {
		port = "8884"
	}

	s.grpcServer = grpc.NewServer()
	s.srv = server.NewServer(logger, time.Now().UnixNano())
	server.RegisterTargetServiceServer(s.grpcServer, s.srv)

	reflection.Register(s.grpcServer)
	s.grpcAddress = ":" + port
	l, err := net.Listen("tcp", s.grpcAddress)
	s.NoError(err)

	go func() {
		err = s.grpcServer.Serve(l)
		s.NoError(err)
	}()
}

func (s *GunSuite) SetupTest() {
	_, err := s.srv.Reset(context.Background(), nil)
	s.NoError(err)
}

func (s *GunSuite) TearDownSuite() {
	s.grpcServer.Stop()
}

func (s *GunSuite) Test_Scenario() {
	ctx := context.Background()
	log := zap.NewNop()
	cfg := grpcscenario.GunConfig{
		Target:      s.grpcAddress,
		Timeout:     0,
		TLS:         false,
		DialOptions: grpcscenario.GrpcDialOptions{},
		AnswLog:     grpcscenario.AnswLogConfig{},
	}
	g := grpcscenario.NewGun(cfg)

	sharedDeps, err := g.WarmUp(&warmup.Options{Log: log, Ctx: ctx})
	s.NoError(err)

	gunDeps := core.GunDeps{Ctx: ctx, Log: log, PoolID: "test", InstanceID: 1, Shared: sharedDeps}
	aggr := &Aggregator{}
	err = g.Bind(aggr, gunDeps)
	s.NoError(err)

	am := &grpcscenario.Scenario{
		Name: "",
		Calls: []grpcscenario.Call{
			{
				Name:           "auth",
				Preprocessors:  nil,
				Postprocessors: nil,
				Tag:            "auth",
				Call:           "target.TargetService.Auth",
				Metadata:       map[string]string{"metadata": "server.proto"},
				Payload:        []byte(`{"login": "1", "pass": "1"}`),
				Sleep:          0,
			},
			{
				Name:           "list",
				Preprocessors:  nil,
				Postprocessors: nil,
				Tag:            "list",
				Call:           "target.TargetService.List",
				Metadata:       map[string]string{"metadata": "server.proto"},
				Payload:        []byte(`{"user_id": {{.request.auth.postprocessor.userId}}, "token": "{{.request.auth.postprocessor.token}}"}`),
				Sleep:          0,
			},
			{
				Name:           "order",
				Preprocessors:  nil,
				Postprocessors: nil,
				Tag:            "order",
				Call:           "target.TargetService.Order",
				Metadata:       map[string]string{"metadata": "server.proto"},
				Payload:        []byte(`{"user_id": {{.request.auth.postprocessor.userId}}, "item_id": 1098, "token": "{{.request.auth.postprocessor.token}}"}`),
				Sleep:          0,
			},
		},
	}

	g.Shoot(am)
	s.Len(aggr.samples, 3)
}

func (s *GunSuite) Test_FullScenario() {
	ctx := context.Background()
	log := zap.NewNop()
	gunConfig := grpcscenario.GunConfig{
		Target:      s.grpcAddress,
		Timeout:     0,
		TLS:         false,
		DialOptions: grpcscenario.GrpcDialOptions{},
		AnswLog:     grpcscenario.AnswLogConfig{},
	}
	g := grpcscenario.NewGun(gunConfig)

	sharedDeps, err := g.WarmUp(&warmup.Options{Log: log, Ctx: ctx})
	s.NoError(err)

	gunDeps := core.GunDeps{Ctx: ctx, Log: log, PoolID: "pool_id", InstanceID: 1, Shared: sharedDeps}
	aggr := &Aggregator{}
	err = g.Bind(aggr, gunDeps)
	s.NoError(err)

	pr, err := ammo.NewProvider(s.fs, scenario.ProviderConfig{File: "testdata/grpc_payload.hcl"})
	require.NoError(s.T(), err)
	go func() {
		_ = pr.Run(ctx, core.ProviderDeps{Log: log, PoolID: "pool_id"})
	}()

	for i := 0; i < 3; i++ {
		am, ok := pr.Acquire()
		s.True(ok)
		g.Shoot(am)
	}
	s.Len(aggr.samples, 15)
	stats, err := s.srv.Stats(context.Background(), nil)
	require.NoError(s.T(), err)

	s.Assert().Equal(map[int64]uint64{1: 1, 2: 1, 3: 1}, stats.Auth.Code200)
	s.Assert().Equal(uint64(0), stats.Auth.Code400)
	s.Assert().Equal(uint64(0), stats.Auth.Code500)
	s.Assert().Equal(map[int64]uint64{1: 1, 2: 1, 3: 1}, stats.List.Code200)
	s.Assert().Equal(uint64(0), stats.List.Code400)
	s.Assert().Equal(uint64(0), stats.List.Code500)
	s.Assert().Equal(map[int64]uint64{1: 3, 2: 3, 3: 3}, stats.Order.Code200)
	s.Assert().Equal(uint64(0), stats.Order.Code400)
	s.Assert().Equal(uint64(0), stats.Order.Code500)

}

func (s *GunSuite) Test_ErrorScenario() {
	ctx := context.Background()
	log := zap.NewNop()
	cfg := grpcscenario.GunConfig{
		Target:      s.grpcAddress,
		Timeout:     0,
		TLS:         false,
		DialOptions: grpcscenario.GrpcDialOptions{},
		AnswLog:     grpcscenario.AnswLogConfig{},
	}
	g := grpcscenario.NewGun(cfg)

	sharedDeps, err := g.WarmUp(&warmup.Options{Log: log, Ctx: ctx})
	s.NoError(err)

	gunDeps := core.GunDeps{Ctx: ctx, Log: log, PoolID: "test", InstanceID: 1, Shared: sharedDeps}
	aggr := &Aggregator{}
	err = g.Bind(aggr, gunDeps)
	s.NoError(err)

	am := &grpcscenario.Scenario{
		Name: "",
		Calls: []grpcscenario.Call{
			{
				Name:          "auth",
				Preprocessors: nil,
				Postprocessors: []grpcscenario.Postprocessor{&postprocessor.AssertResponse{
					Payload:    nil,
					StatusCode: 200,
				}},
				Tag:      "auth",
				Call:     "target.TargetService.Auth",
				Metadata: map[string]string{"metadata": "server.proto"},
				Payload:  []byte(`{"login": "1", "pass": "2"}`), // invalid pass
				Sleep:    0,
			},
			{
				Name:           "list",
				Preprocessors:  nil,
				Postprocessors: nil,
				Tag:            "list",
				Call:           "target.TargetService.List",
				Metadata:       map[string]string{"metadata": "server.proto"},
				Payload:        []byte(`{"user_id": {{.request.auth.postprocessor.userId}}, "token": "{{.request.auth.postprocessor.token}}"}`),
				Sleep:          0,
			},
			{
				Name:           "order",
				Preprocessors:  nil,
				Postprocessors: nil,
				Tag:            "order",
				Call:           "target.TargetService.Order",
				Metadata:       map[string]string{"metadata": "server.proto"},
				Payload:        []byte(`{"user_id": {{.request.auth.postprocessor.userId}}, "item_id": 1098, "token": "{{.request.auth.postprocessor.token}}"}`),
				Sleep:          0,
			},
		},
	}

	g.Shoot(am)
	s.Len(aggr.samples, 1)
}

type Aggregator struct {
	samples []core.Sample
}

func (a *Aggregator) Run(ctx context.Context, deps core.AggregatorDeps) error {
	return nil
}

func (a *Aggregator) Report(s core.Sample) {
	a.samples = append(a.samples, s)
}
