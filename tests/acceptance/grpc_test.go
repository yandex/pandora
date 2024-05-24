package acceptance

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/yandex/pandora/cli"
	grpcimport "github.com/yandex/pandora/components/grpc/import"
	phttpimport "github.com/yandex/pandora/components/phttp/import"
	"github.com/yandex/pandora/core/engine"
	coreimport "github.com/yandex/pandora/core/import"
	"github.com/yandex/pandora/examples/grpc/server"
	"github.com/yandex/pandora/lib/pointer"
	"github.com/yandex/pandora/lib/testutil"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	pandoraLogger := testutil.NewNullLogger()
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

		conf := parseFileContentToCliConfig(t, baseFile, nil)

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

		conf := parseFileContentToCliConfig(t, baseFile, func(c *PandoraConfigGRPC) {
			c.Pools[0].Gun.ReflectPort = pointer.ToInt64(18889)
		})

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

	t.Run("reflect with custom metadata", func(t *testing.T) {
		metadataKey := "testKey"
		metadataValue := "testValue"
		wrongMDValuesLengthError := errors.New("wrong metadata values length")
		wrongMDValueError := errors.New("wrong metadata value")
		metadataChecker := func(ctx context.Context) (context.Context, error) {
			md, ok := metadata.FromIncomingContext(ctx)
			if !ok {
				return nil, wrongMDValuesLengthError
			}
			vals := md.Get(metadataKey)
			if len(vals) != 1 {
				return nil, wrongMDValuesLengthError
			}
			if vals[0] != metadataValue {
				return nil, wrongMDValueError
			}
			return ctx, nil
		}
		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(MetadataServerInterceptor(metadataChecker)),
			grpc.StreamInterceptor(MetadataServerStreamInterceptor(metadataChecker)))
		srv := server.NewServer(logger, time.Now().UnixNano())
		server.RegisterTargetServiceServer(grpcServer, srv)
		grpcAddress := "localhost:18888"
		reflection.Register(grpcServer)
		l, err := net.Listen("tcp", grpcAddress)
		require.NoError(t, err)
		go func() {
			err = grpcServer.Serve(l)
			require.NoError(t, err)
		}()

		defer func() {
			grpcServer.Stop()
		}()

		cases := []struct {
			name string
			conf *cli.CliConfig
			err  error
		}{
			{
				name: "success",
				conf: parseFileContentToCliConfig(t, baseFile, func(c *PandoraConfigGRPC) {
					c.Pools[0].Gun.ReflectMetadata = map[string]string{metadataKey: metadataValue}
				}),
			},
			{
				name: "no metadata",
				conf: parseFileContentToCliConfig(t, baseFile, nil),
				err:  wrongMDValuesLengthError,
			},
			{
				name: "wrong metadata value",
				conf: parseFileContentToCliConfig(t, baseFile, func(c *PandoraConfigGRPC) {
					c.Pools[0].Gun.ReflectMetadata = map[string]string{metadataKey: "wrong-value"}
				}),
				err: wrongMDValueError,
			},
		}

		for _, cc := range cases {
			t.Run(cc.name, func(t *testing.T) {
				require.Equal(t, 1, len(cc.conf.Engine.Pools))
				aggr := &aggregator{}
				cc.conf.Engine.Pools[0].Aggregator = aggr

				pandora := engine.New(pandoraLogger, pandoraMetrics, cc.conf.Engine)
				err = pandora.Run(context.Background())

				if cc.err == nil {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
					require.Contains(t, err.Error(), cc.err.Error())
				}
			})
		}
	})
}

func TestGrpcGunSuite(t *testing.T) {
	suite.Run(t, new(GrpcGunSuite))
}

type GrpcGunSuite struct {
	suite.Suite
	fs      afero.Fs
	log     *zap.Logger
	metrics engine.Metrics
}

func (s *GrpcGunSuite) SetupSuite() {
	s.fs = afero.NewOsFs()
	testOnce.Do(func() {
		coreimport.Import(s.fs)
		phttpimport.Import(s.fs)
		grpcimport.Import(s.fs)
	})

	s.log = testutil.NewNullLogger()
	s.metrics = newEngineMetrics("grpc_suite")
}

func (s *GrpcGunSuite) Test_Run() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	baseFile, err := os.ReadFile("testdata/grpc/base.yaml")
	s.Require().NoError(err)

	tests := []struct {
		name      string
		overwrite func(c *PandoraConfigGRPC)
		wantCnt   int64
	}{
		{
			name:    "default testdata/grpc/base.yaml",
			wantCnt: 8,
		},
		{
			name: "add pool-size testdata/grpc/base.yaml",
			overwrite: func(c *PandoraConfigGRPC) {
				c.Pools[0].Gun.SharedClient.Enabled = true
				c.Pools[0].Gun.SharedClient.ClientNumber = 2
			},
			wantCnt: 8,
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {

			grpcServer := grpc.NewServer()
			srv := server.NewServer(logger, time.Now().UnixNano())
			server.RegisterTargetServiceServer(grpcServer, srv)
			reflection.Register(grpcServer)
			l, err := net.Listen("tcp", ":18888")
			s.Require().NoError(err)
			go func() {
				err = grpcServer.Serve(l)
				s.Require().NoError(err)
			}()
			defer func() {
				grpcServer.Stop()
			}()

			conf := parseFileContentToCliConfig(s.T(), baseFile, tt.overwrite)

			aggr := &aggregator{}
			conf.Engine.Pools[0].Aggregator = aggr
			pandora := engine.New(s.log, s.metrics, conf.Engine)

			err = pandora.Run(context.Background())
			s.Require().NoError(err)
			stats, err := srv.Stats(context.Background(), nil)
			s.Require().NoError(err)
			s.Require().Equal(tt.wantCnt, stats.Hello)
		})
	}
}

func parseFileContentToCliConfig(t *testing.T, baseFile []byte, overwrite func(c *PandoraConfigGRPC)) *cli.CliConfig {
	cfg := PandoraConfigGRPC{}
	err := yaml.Unmarshal(baseFile, &cfg)
	require.NoError(t, err)
	if overwrite != nil {
		overwrite(&cfg)
	}
	b, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	mapCfg := map[string]any{}
	err = yaml.Unmarshal(b, &mapCfg)
	require.NoError(t, err)

	return decodeConfig(t, mapCfg)
}

func MetadataServerInterceptor(metadataChecker func(ctx context.Context) (context.Context, error)) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		ctx, err = metadataChecker(ctx)
		if err != nil {
			return nil, fmt.Errorf("metadata checker: %w", err)
		}
		return handler(ctx, req)
	}
}

func MetadataServerStreamInterceptor(metadataChecker func(ctx context.Context) (context.Context, error)) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		ctx := ss.Context()
		_, err = metadataChecker(ctx)
		if err != nil {
			return fmt.Errorf("metadata checker: %w", err)
		}
		return handler(srv, ss)
	}
}
