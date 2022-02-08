package grpc

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/yandex/pandora/core/warmup"
	"google.golang.org/grpc/credentials"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Ammo struct {
	Tag      string                 `json:"tag"`
	Call     string                 `json:"call"`
	Metadata map[string]string      `json:"metadata"`
	Payload  map[string]interface{} `json:"payload"`
}

type Sample struct {
	URL              string
	ShootTimeSeconds float64
}

type GunConfig struct {
	Target  string        `validate:"required"`
	Timeout time.Duration `config:"timeout"` // grpc request timeout
	TLS     bool
}

type Gun struct {
	DebugLog bool
	client   *grpc.ClientConn
	conf     GunConfig
	aggr     core.Aggregator
	core.GunDeps

	stub     grpcdynamic.Stub
	services map[string]desc.MethodDescriptor
}

func (g *Gun) WarmUp(opts *warmup.Options) (interface{}, error) {
	conn, err := makeGRPCConnect(g.conf.Target, g.conf.TLS)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to target: %w", err)
	}
	defer conn.Close()

	meta := make(metadata.MD)
	refCtx := metadata.NewOutgoingContext(context.Background(), meta)
	refClient := grpcreflect.NewClient(refCtx, reflectpb.NewServerReflectionClient(conn))
	listServices, err := refClient.ListServices()
	if err != nil {
		opts.Log.Fatal("Fatal: failed to get services list\n %s\n", zap.Error(err))
	}
	services := make(map[string]desc.MethodDescriptor)
	for _, s := range listServices {
		service, err := refClient.ResolveService(s)
		if err != nil {
			if grpcreflect.IsElementNotFoundError(err) {
				continue
			}
			opts.Log.Fatal("FATAL ResolveService: %s", zap.Error(err))
		}
		listMethods := service.GetMethods()
		for _, m := range listMethods {
			services[m.GetFullyQualifiedName()] = *m
		}
	}
	return services, nil
}

func (g *Gun) AcceptWarmUpResult(i interface{}) error {
	services, ok := i.(map[string]desc.MethodDescriptor)
	if !ok {
		return fmt.Errorf("grpc WarmUp result should be services: map[string]desc.MethodDescriptor")
	}
	g.services = services
	return nil
}

func NewGun(conf GunConfig) *Gun {
	return &Gun{conf: conf}
}

func (g *Gun) Bind(aggr core.Aggregator, deps core.GunDeps) error {
	conn, err := makeGRPCConnect(g.conf.Target, g.conf.TLS)
	if err != nil {
		log.Fatalf("FATAL: grpc.Dial failed\n %s\n", err)
	}
	g.client = conn
	g.aggr = aggr
	g.GunDeps = deps
	g.stub = grpcdynamic.NewStub(conn)

	if ent := deps.Log.Check(zap.DebugLevel, "Gun bind"); ent != nil {
		// Enable debug level logging during shooting. Creating log entries isn't free.
		g.DebugLog = true
	}

	return nil
}

func (g *Gun) Shoot(ammo core.Ammo) {
	customAmmo := ammo.(*Ammo)
	g.shoot(customAmmo)
}

func (g *Gun) shoot(ammo *Ammo) {

	code := 0
	sample := netsample.Acquire(ammo.Tag)
	defer func() {
		sample.SetProtoCode(code)
		g.aggr.Report(sample)
	}()

	method, ok := g.services[ammo.Call]
	if !ok {
		log.Fatalf("Fatal: No such method %s\n", ammo.Call)
		return
	}

	payloadJSON, err := json.Marshal(ammo.Payload)
	if err != nil {
		log.Fatalf("FATAL: Payload parsing error %s\n", err)
		return
	}

	md := method.GetInputType()
	message := dynamic.NewMessage(md)
	err = message.UnmarshalJSON(payloadJSON)
	if err != nil {
		code = 400
		log.Printf("BAD REQUEST: %s\n", err)
		return
	}

	meta := make(metadata.MD)
	if ammo.Metadata != nil && len(ammo.Metadata) > 0 {
		for key, value := range ammo.Metadata {
			meta = metadata.Pairs(key, value)
		}
	}

	timeout := time.Second * 15
	if g.conf.Timeout != 0 {
		timeout = time.Second * g.conf.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, meta)
	out, err := g.stub.InvokeRpc(ctx, &method, message)
	code = convertGrpcStatus(err)

	if err != nil {
		log.Printf("Response error: %s\n", err)
	}

	if g.DebugLog {
		g.Log.Debug("Request:", zap.Stringer("method", &method), zap.Stringer("message", message))
		g.Log.Debug("Response:", zap.Stringer("resp", out))
	}

}

func makeGRPCConnect(target string, isTLS bool) (conn *grpc.ClientConn, err error) {
	opts := []grpc.DialOption{}
	if isTLS {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithTimeout(time.Second))
	opts = append(opts, grpc.WithUserAgent("load test, pandora universal grpc shooter"))

	conn, err = grpc.Dial(target, opts...)
	return
}

func convertGrpcStatus(err error) int {
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
