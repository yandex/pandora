package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/jhump/protoreflect/dynamic"
	grpcgun "github.com/yandex/pandora/components/guns/grpc"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/warmup"
	"github.com/yandex/pandora/lib/answlog"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/metadata"
)

const defaultTimeout = time.Second * 15

type GunConfig struct {
	Target          string            `validate:"required"`
	ReflectPort     int64             `config:"reflect_port"`
	ReflectMetadata map[string]string `config:"reflect_metadata"`
	Timeout         time.Duration     `config:"timeout"` // grpc request timeout
	TLS             bool              `config:"tls"`
	DialOptions     GrpcDialOptions   `config:"dial_options"`
	AnswLog         AnswLogConfig     `config:"answlog"`
}

type GrpcDialOptions struct {
	Authority string        `config:"authority"`
	Timeout   time.Duration `config:"timeout"`
}

type AnswLogConfig struct {
	Enabled bool   `config:"enabled"`
	Path    string `config:"path"`
	Filter  string `config:"filter" valid:"oneof=all warning error"`
}

func DefaultGunConfig() GunConfig {
	return GunConfig{
		Target: "default target",
		AnswLog: AnswLogConfig{
			Enabled: false,
			Path:    "answ.log",
			Filter:  "all",
		},
	}
}

func NewGun(conf GunConfig) *Gun {
	answLog := answlog.Init(conf.AnswLog.Path, conf.AnswLog.Enabled)
	r := rand.New(rand.NewSource(0)) //TODO: use real random
	return &Gun{
		templ: NewTextTemplater(),
		gun: &grpcgun.Gun{Conf: grpcgun.GunConfig{
			Target:          conf.Target,
			ReflectPort:     conf.ReflectPort,
			ReflectMetadata: conf.ReflectMetadata,
			Timeout:         conf.Timeout,
			TLS:             conf.TLS,
			DialOptions: grpcgun.GrpcDialOptions{
				Authority: conf.DialOptions.Authority,
				Timeout:   conf.DialOptions.Timeout,
			},
			AnswLog: grpcgun.AnswLogConfig{
				Enabled: conf.AnswLog.Enabled,
				Path:    conf.AnswLog.Path,
				Filter:  conf.AnswLog.Filter,
			},
		},
			AnswLog: answLog},
		rand: r,
	}
}

type Gun struct {
	gun   *grpcgun.Gun
	rand  *rand.Rand
	templ Templater
}

func (g *Gun) WarmUp(opts *warmup.Options) (interface{}, error) {
	return g.gun.WarmUp(opts)
}

func (g *Gun) Bind(aggr core.Aggregator, deps core.GunDeps) error {
	return g.gun.Bind(aggr, deps)
}

func (g *Gun) Shoot(am core.Ammo) {
	scen := am.(*Scenario)

	templateVars := map[string]any{}
	if scen.VariableStorage != nil {
		templateVars["source"] = scen.VariableStorage.Variables()
	} else {
		templateVars["source"] = map[string]any{}
	}

	err := g.shoot(scen, templateVars)
	if err != nil {
		g.gun.Log.Warn("Invalid ammo", zap.Uint64("request", scen.id), zap.Error(err))
		return
	} else {
		g.gun.Log.Debug("Valid ammo", zap.Uint64("request", scen.id))
	}
}

func (g *Gun) shoot(ammo *Scenario, templateVars map[string]any) error {
	if templateVars == nil {
		templateVars = map[string]any{}
	}

	requestVars := map[string]any{}
	templateVars["request"] = requestVars
	if g.gun.DebugLog {
		g.gun.GunDeps.Log.Debug("Source variables", zap.Any("variables", templateVars))
	}

	startAt := time.Now()
	for _, call := range ammo.Calls {
		tag := ammo.Name + "." + call.Tag
		sample := netsample.Acquire(tag)

		err := g.shootStep(&call, sample, ammo.Name, templateVars, requestVars)
		if err != nil {
			return err
		}
	}
	spent := time.Since(startAt)
	if ammo.MinWaitingTime > spent {
		time.Sleep(ammo.MinWaitingTime - spent)
	}
	return nil
}

func (g *Gun) shootStep(step *Call, sample *netsample.Sample, ammoName string, templateVars map[string]any, requestVars map[string]any) error {
	const op = "base_gun.shootStep"
	code := 0
	defer func() {
		sample.SetProtoCode(code)
		g.gun.Aggr.Report(sample)
	}()

	stepVars := map[string]any{}
	requestVars[step.Name] = stepVars

	// Preprocessor
	preprocVars := map[string]any{}
	for _, preProcessor := range step.Preprocessors {
		pp, err := preProcessor.Process(step, templateVars)
		if err != nil {
			return fmt.Errorf("%s preProcessor %w", op, err)
		}
		preprocVars = mergeMaps(preprocVars, pp)
		if g.gun.DebugLog {
			g.gun.GunDeps.Log.Debug("PreparePreprocessor variables", zap.Any(fmt.Sprintf(".request.%s.preprocessor", step.Name), pp))
		}
	}
	stepVars["preprocessor"] = preprocVars

	// Template
	payloadJSON, err := g.templ.Apply(step.Payload, step.Metadata, templateVars, ammoName, step.Name)
	if err != nil {
		return fmt.Errorf("%s templater.Apply %w", op, err)
	}

	// Method
	method, ok := g.gun.Services[step.Call]
	if !ok {
		g.gun.GunDeps.Log.Error("invalid step.Call", zap.String("method", step.Call),
			zap.Strings("allowed_methods", maps.Keys(g.gun.Services)))
		return fmt.Errorf("%s invalid step.Call", op)
	}

	md := method.GetInputType()
	message := dynamic.NewMessage(md)
	err = message.UnmarshalJSON(payloadJSON)
	if err != nil {
		code = 400
		g.gun.GunDeps.Log.Error("invalid payload. Cant unmarshal gRPC", zap.Error(err))
		return fmt.Errorf("%s invalid payload. Cant unmarshal gRPC", op)
	}

	timeout := defaultTimeout
	if g.gun.Conf.Timeout != 0 {
		timeout = g.gun.Conf.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	ctx = metadata.NewOutgoingContext(ctx, metadata.New(step.Metadata))
	out, grpcErr := g.gun.Stub.InvokeRpc(ctx, &method, message)
	code = grpcgun.ConvertGrpcStatus(grpcErr)
	sample.SetProtoCode(code) // for setRTT inside

	if grpcErr != nil {
		g.gun.GunDeps.Log.Error("response error", zap.Error(err))
	}

	g.gun.Answ(&method, message, step.Metadata, out, grpcErr, code)

	for _, postProcessor := range step.Postprocessors {
		pp, err := postProcessor.Process(out, code)
		if err != nil {
			return fmt.Errorf("%s postProcessor %w", op, err)
		}
		stepVars = mergeMaps(stepVars, pp)
		if g.gun.DebugLog {
			g.gun.GunDeps.Log.Debug("Postprocessor variables", zap.Any(fmt.Sprintf(".request.%s.postprocessor", step.Name), pp))
		}
	}
	if out != nil {
		// Postprocessor
		// if it is nessesary
		md = method.GetOutputType()
		message = dynamic.NewMessage(md)
		err = message.ConvertFrom(out)
		if err != nil {
			// unexpected result
			return fmt.Errorf("%s message.ConvertFrom `%s`; err: %w", op, out.String(), err)

		}
		b, err := message.MarshalJSON()
		if err != nil {
			// unexpected result
			return fmt.Errorf("%s message.MarshalJSON %w", op, err)
		}
		var outMap map[string]any
		err = json.Unmarshal(b, &outMap)
		if err != nil {
			// unexpected result
			return fmt.Errorf("%s json.Unmarshal %w", op, err)
		}
		stepVars["postprocessor"] = outMap

		if g.gun.DebugLog {
			g.gun.GunDeps.Log.Debug("Postprocessor variables", zap.String(fmt.Sprintf(".resuest.%s.postprocessor", step.Name), out.String()))
		}
	}

	if step.Sleep > 0 {
		time.Sleep(step.Sleep)
	}

	return nil
}

// mergeMaps merges newvars into previous
// if key exists in previous, it will be skipped
func mergeMaps(previous map[string]any, newvars map[string]any) map[string]any {
	for k, v := range newvars {
		if _, ok := previous[k]; !ok {
			previous[k] = v
		}
	}
	return previous
}

func (g *Gun) reportErr(sample *netsample.Sample, err error) {
	if err == nil {
		return
	}
	sample.AddTag("__EMPTY__")
	sample.SetProtoCode(0)
	sample.SetErr(err)
	g.gun.Aggr.Report(sample)
}

var _ warmup.WarmedUp = (*Gun)(nil)
