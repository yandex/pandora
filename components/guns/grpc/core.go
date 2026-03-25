package grpc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/yandex/pandora/components/answ/filter"
	"github.com/yandex/pandora/components/answ/sampler"
	"github.com/yandex/pandora/components/providers/grpc/ammo"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/clientpool"
	"github.com/yandex/pandora/core/warmup"
	"github.com/yandex/pandora/lib/answlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const defaultTimeout = time.Second * 15

type Sample struct {
	URL              string
	ShootTimeSeconds float64
}

type GrpcDialOptions struct {
	Authority string        `config:"authority"`
	Timeout   time.Duration `config:"timeout"`
}

type GunConfig struct {
	Target          string            `validate:"required"`
	ReflectPort     int64             `config:"reflect_port"`
	ReflectMetadata map[string]string `config:"reflect_metadata"`
	Timeout         time.Duration     `config:"timeout"` // grpc request timeout
	TLS             bool              `config:"tls"`
	DialOptions     GrpcDialOptions   `config:"dial_options"`
	AnswLog         answlog.Config    `config:"answlog"`
	SharedClient    struct {
		ClientNumber int  `config:"client-number,omitempty"`
		Enabled      bool `config:"enabled"`
	} `config:"shared-client,omitempty"`
}

type Gun struct {
	DebugLog bool
	Conf     GunConfig
	Aggr     core.Aggregator
	core.GunDeps

	Stub     grpcdynamic.Stub
	Services map[string]desc.MethodDescriptor

	AnswLog *answlog.Logger
}

func DefaultGunConfig() GunConfig {
	return GunConfig{
		Target: "default target",
		AnswLog: answlog.Config{
			Enabled: false,
			Filter:  filter.FilterAll,
			Sampling: answlog.Sampling{
				Enabled: true,
				Pattern: sampler.FactorPattern{
					Factor: 10,
				},
			},
			Masking: &answlog.PassAllMasker{},
		},
	}
}

func (g *Gun) WarmUp(opts *warmup.Options) (any, error) {
	return g.createSharedDeps(opts)
}

func (g *Gun) createSharedDeps(opts *warmup.Options) (*SharedDeps, error) {
	services, err := g.prepareMethodList(opts)
	if err != nil {
		return nil, err
	}
	clientPool, err := g.prepareClientPool()
	if err != nil {
		return nil, err
	}
	return &SharedDeps{
		services:   services,
		clientPool: clientPool,
		answlog: answlog.Init(
			g.Conf.AnswLog.Path,
			g.Conf.AnswLog.Enabled,
			answlog.WithFilter(filter.NewGRPCStatusCodeFilter(g.Conf.AnswLog.Filter)),
			answlog.WithSampler(sampler.NewStatusCodeSampler(g.Conf.AnswLog.Sampling.Pattern, 20), g.Conf.AnswLog.Sampling.Enabled),
			answlog.WithMasker(g.Conf.AnswLog.Masking),
		),
	}, nil
}

func (g *Gun) prepareMethodList(opts *warmup.Options) (map[string]desc.MethodDescriptor, error) {
	conn, err := g.makeReflectionConnect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target: %w", err)
	}
	defer conn.Close()

	refCtx := metadata.NewOutgoingContext(context.Background(), metadata.New(g.Conf.ReflectMetadata))
	refClient := grpcreflect.NewClientAuto(refCtx, conn)
	listServices, err := refClient.ListServices()
	if err != nil {
		opts.Log.Error("failed to get services list", zap.Error(err)) // WarmUp calls before Bind()
		return nil, fmt.Errorf("refClient.ListServices err: %w", err)
	}
	services := make(map[string]desc.MethodDescriptor)
	for _, s := range listServices {
		service, err := refClient.ResolveService(s)
		if err != nil {
			if grpcreflect.IsElementNotFoundError(err) {
				continue
			}
			opts.Log.Error("cant resolveService", zap.String("service_name", s), zap.Error(err)) // WarmUp calls before Bind()
			return nil, fmt.Errorf("cant resolveService %s; err: %w", s, err)
		}
		listMethods := service.GetMethods()
		for _, m := range listMethods {
			// package.Service.Method
			services[m.GetFullyQualifiedName()] = *m
			// package.Service/Method
			services[m.GetParent().GetFullyQualifiedName()+"/"+m.GetName()] = *m
			// /package.Service/Method
			services["/"+m.GetParent().GetFullyQualifiedName()+"/"+m.GetName()] = *m
		}
	}
	return services, nil
}

func (g *Gun) prepareClientPool() (*clientpool.Pool[grpcdynamic.Stub], error) {
	if !g.Conf.SharedClient.Enabled {
		return nil, nil
	}
	if g.Conf.SharedClient.ClientNumber < 1 {
		g.Conf.SharedClient.ClientNumber = 1
	}
	clientPool, err := clientpool.New[grpcdynamic.Stub](g.Conf.SharedClient.ClientNumber)
	if err != nil {
		return nil, fmt.Errorf("create clientpool err: %w", err)
	}
	for i := 0; i < g.Conf.SharedClient.ClientNumber; i++ {
		conn, err := g.makeConnect()
		if err != nil {
			return nil, fmt.Errorf("makeGRPCConnect fail %w", err)
		}
		clientPool.Add(grpcdynamic.NewStub(conn))
	}
	return clientPool, nil
}

func NewGun(conf GunConfig) *Gun {
	return &Gun{Conf: conf}
}

func (g *Gun) Bind(aggr core.Aggregator, deps core.GunDeps) error {
	sharedDeps, ok := deps.Shared.(*SharedDeps)
	if !ok {
		return errors.New("grpc WarmUp result should be struct: *SharedDeps")
	}
	g.Services = sharedDeps.services
	if sharedDeps.clientPool != nil {
		g.Stub = sharedDeps.clientPool.Next()
	} else {
		conn, err := g.makeConnect()
		if err != nil {
			return fmt.Errorf("makeGRPCConnect fail %w", err)
		}
		g.Stub = grpcdynamic.NewStub(conn)
	}
	if sharedDeps.answlog != nil {
		g.AnswLog = sharedDeps.answlog
	} else if g.AnswLog == nil {
		g.AnswLog = answlog.NewNop()
	}

	g.Aggr = aggr
	g.GunDeps = deps

	if ent := deps.Log.Check(zap.DebugLevel, "Gun bind"); ent != nil {
		g.DebugLog = true
	}

	return nil
}

func (g *Gun) Shoot(am core.Ammo) {
	customAmmo := am.(*ammo.Ammo)
	g.shoot(customAmmo)
}

func (g *Gun) shoot(ammo *ammo.Ammo) {
	code := 0
	sample := netsample.Acquire(ammo.Tag)
	defer func() {
		sample.SetProtoCode(code)
		g.Aggr.Report(sample)
	}()

	method, ok := g.Services[ammo.Call]
	if !ok {
		g.GunDeps.Log.Error("invalid ammo.Call", zap.String("method", ammo.Call),
			zap.Strings("allowed_methods", maps.Keys(g.Services)))
		return
	}

	payloadJSON, err := json.Marshal(ammo.Payload)
	if err != nil {
		g.GunDeps.Log.Error("invalid payload. Cant unmarshal json", zap.Error(err))
		return
	}
	md := method.GetInputType()
	message := dynamic.NewMessage(md)
	err = message.UnmarshalJSON(payloadJSON)
	if err != nil {
		code = 400
		g.GunDeps.Log.Error("invalid payload. Cant unmarshal gRPC", zap.Error(err))
		return
	}

	timeout := defaultTimeout
	if g.Conf.Timeout != 0 {
		timeout = g.Conf.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(ammo.Metadata))
	out, grpcErr := g.Stub.InvokeRpc(ctx, &method, message)
	code = ConvertGrpcStatus(grpcErr)

	if grpcErr != nil {
		g.GunDeps.Log.Error("response error", zap.Error(grpcErr))
	}

	g.AnswLogging(&method, message, ammo.Metadata, out, grpcErr)
}

func (g *Gun) AnswLogging(method *desc.MethodDescriptor, request proto.Message, metadata map[string]string, response proto.Message, grpcErr error) {
	grpcCode := int(status.Convert(grpcErr).Code())
	g.AnswLog.Report("REQUEST/RESPONSE", []zapcore.Field{zap.Int(answlog.FilterAndSampleGroup, grpcCode)}, func() []zapcore.Field {
		var respField zapcore.Field

		if response != nil {
			respField = zap.Stringer("resp", response)
		} else {
			respField = zap.String("resp", "empty")
		}

		return []zapcore.Field{zap.Stringer("method", method), zap.Stringer("req", request), zap.Any("metadata", metadata), respField, zap.Error(grpcErr)}
	})
}

func (g *Gun) makeConnect() (conn *grpc.ClientConn, err error) {
	return MakeGRPCConnect(g.Conf.Target, g.Conf.TLS, g.Conf.DialOptions)
}

func (g *Gun) makeReflectionConnect() (conn *grpc.ClientConn, err error) {
	target := replacePort(g.Conf.Target, g.Conf.ReflectPort)
	return MakeGRPCConnect(target, g.Conf.TLS, g.Conf.DialOptions)
}

func MakeGRPCConnect(target string, isTLS bool, dialOptions GrpcDialOptions) (conn *grpc.ClientConn, err error) {
	opts := []grpc.DialOption{}
	if isTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	timeout := time.Second
	if dialOptions.Timeout != 0 {
		timeout = dialOptions.Timeout
	}
	opts = append(opts, grpc.WithUserAgent("load test, pandora universal grpc shooter"))

	if dialOptions.Authority != "" {
		opts = append(opts, grpc.WithAuthority(dialOptions.Authority))
	}

	ctx, cncl := context.WithTimeout(context.Background(), timeout)
	defer cncl()
	return grpc.DialContext(ctx, target, opts...)
}

func ConvertGrpcStatus(err error) int {
	s := status.Convert(err)

	switch s.Code() {
	case codes.OK:
		return 200
	case codes.Canceled:
		return 499
	case codes.InvalidArgument:
		return 400
	case codes.DeadlineExceeded:
		return 504
	case codes.NotFound:
		return 404
	case codes.AlreadyExists:
		return 409
	case codes.PermissionDenied:
		return 403
	case codes.ResourceExhausted:
		return 429
	case codes.FailedPrecondition:
		return 400
	case codes.Aborted:
		return 409
	case codes.OutOfRange:
		return 400
	case codes.Unimplemented:
		return 501
	case codes.Unavailable:
		return 503
	case codes.Unauthenticated:
		return 401
	default:
		return 500
	}
}

func replacePort(host string, port int64) string {
	if port == 0 {
		return host
	}
	split := strings.Split(host, ":")
	if len(split) == 1 {
		return host + ":" + strconv.FormatInt(port, 10)
	}

	oldPort := split[len(split)-1]
	if _, err := strconv.ParseInt(oldPort, 10, 64); err != nil {
		return host + ":" + strconv.FormatInt(port, 10)
	}

	split[len(split)-1] = strconv.FormatInt(port, 10)
	return strings.Join(split, ":")
}

var _ warmup.WarmedUp = (*Gun)(nil)
