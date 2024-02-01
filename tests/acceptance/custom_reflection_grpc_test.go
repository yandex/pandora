package acceptance

import (
	"context"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	grpcimport "github.com/yandex/pandora/components/grpc/import"
	phttpimport "github.com/yandex/pandora/components/phttp/import"
	"github.com/yandex/pandora/core/engine"
	coreimport "github.com/yandex/pandora/core/import"
	"github.com/yandex/pandora/examples/grpc/server"
	"github.com/yandex/pandora/lib/pointer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gopkg.in/yaml.v2"
)

func TestCheckGRPCReflectServer(t *testing.T) {
	fs := afero.NewOsFs()
	testOnce.Do(func() {
		coreimport.Import(fs)
		phttpimport.Import(fs)
		grpcimport.Import(fs)
	})
	pandoraLogger := newNullLogger()
	pandoraMetrics := newEngineMetrics("reflect")
	baseFile, err := os.ReadFile("testdata/grpc/base.yaml")
	require.NoError(t, err)

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	t.Run("reflect not found", func(t *testing.T) {
		grpcServer := grpc.NewServer()
		srv := server.NewServer(logger, time.Now().UnixNano())
		server.RegisterTargetServiceServer(grpcServer, srv)
		grpcAddress := ":18888"
		// Don't register reflection handler
		// reflection.Register(grpcServer)
		l, err := net.Listen("tcp", grpcAddress)
		require.NoError(t, err)
		go func() {
			err = grpcServer.Serve(l)
			require.NoError(t, err)
		}()

		defer func() {
			grpcServer.Stop()
		}()

		cfg := PandoraConfigGRPC{}
		err = yaml.Unmarshal(baseFile, &cfg)
		require.NoError(t, err)
		b, err := yaml.Marshal(cfg)
		require.NoError(t, err)
		mapCfg := map[string]any{}
		err = yaml.Unmarshal(b, &mapCfg)
		require.NoError(t, err)
		conf := decodeConfig(t, mapCfg)

		require.Equal(t, 1, len(conf.Engine.Pools))
		aggr := &aggregator{}
		conf.Engine.Pools[0].Aggregator = aggr
		pandora := engine.New(pandoraLogger, pandoraMetrics, conf.Engine)
		err = pandora.Run(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "gun warm up failed")
		require.Contains(t, err.Error(), "unknown service grpc.reflection.v1alpha.ServerReflection")

		st, err := srv.Stats(context.Background(), nil)
		require.NoError(t, err)
		require.Equal(t, int64(0), st.Hello)
	})

	t.Run("reflect on another port", func(t *testing.T) {
		grpcServer := grpc.NewServer()
		srv := server.NewServer(logger, time.Now().UnixNano())
		server.RegisterTargetServiceServer(grpcServer, srv)
		grpcAddress := ":18888"
		l, err := net.Listen("tcp", grpcAddress)
		require.NoError(t, err)
		go func() {
			err := grpcServer.Serve(l)
			require.NoError(t, err)
		}()

		reflectionGrpcServer := grpc.NewServer()
		reflectionSrv := server.NewServer(logger, time.Now().UnixNano())
		server.RegisterTargetServiceServer(reflectionGrpcServer, reflectionSrv)
		grpcAddress = ":18889"
		reflection.Register(reflectionGrpcServer)
		listenRef, err := net.Listen("tcp", grpcAddress)
		require.NoError(t, err)
		go func() {
			err := reflectionGrpcServer.Serve(listenRef)
			require.NoError(t, err)
		}()
		defer func() {
			grpcServer.Stop()
			reflectionGrpcServer.Stop()
		}()

		cfg := PandoraConfigGRPC{}
		err = yaml.Unmarshal(baseFile, &cfg)
		cfg.Pools[0].Gun.ReflectPort = pointer.ToInt64(18889)
		require.NoError(t, err)
		b, err := yaml.Marshal(cfg)
		require.NoError(t, err)
		mapCfg := map[string]any{}
		err = yaml.Unmarshal(b, &mapCfg)
		require.NoError(t, err)
		conf := decodeConfig(t, mapCfg)

		require.Equal(t, 1, len(conf.Engine.Pools))
		aggr := &aggregator{}
		conf.Engine.Pools[0].Aggregator = aggr
		pandora := engine.New(pandoraLogger, pandoraMetrics, conf.Engine)
		err = pandora.Run(context.Background())
		require.NoError(t, err)

		st, err := srv.Stats(context.Background(), nil)
		require.NoError(t, err)
		require.Equal(t, int64(8), st.Hello)
	})
}
