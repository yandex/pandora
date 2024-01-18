package grpc

import (
	"fmt"
	"time"

	gun "github.com/yandex/pandora/components/guns/grpc/scenario"
	"github.com/yandex/pandora/components/providers/scenario/config"
	"github.com/yandex/pandora/components/providers/scenario/vs"
	"github.com/yandex/pandora/lib/mp"
)

type IteratorIniter interface {
	InitIterator(iter mp.Iterator)
}

func decodeAmmo(cfg *config.AmmoConfig, storage *vs.SourceStorage) ([]*gun.Scenario, error) {
	callRegistry := make(map[string]config.CallConfig, len(cfg.Calls))
	for _, req := range cfg.Calls {
		callRegistry[req.Name] = req
	}

	scenarioRegistry := map[string]config.ScenarioConfig{}
	for _, sc := range cfg.Scenarios {
		scenarioRegistry[sc.Name] = sc
	}

	names, size := config.SpreadNames(cfg.Scenarios)
	result := make([]*gun.Scenario, 0, size)
	for _, sc := range cfg.Scenarios {
		a, err := convertScenarioToAmmo(sc, callRegistry)
		if err != nil {
			return nil, fmt.Errorf("failed to convert scenario %s: %w", sc.Name, err)
		}
		a.VariableStorage = storage
		ns, ok := names[sc.Name]
		if !ok {
			return nil, fmt.Errorf("scenario %s is not found", sc.Name)
		}
		for i := 0; i < ns; i++ {
			result = append(result, a)
		}
	}

	return result, nil
}

func convertScenarioToAmmo(sc config.ScenarioConfig, reqs map[string]config.CallConfig) (*gun.Scenario, error) {
	iter := mp.NewNextIterator(time.Now().UnixNano())
	result := &gun.Scenario{Name: sc.Name, MinWaitingTime: time.Millisecond * time.Duration(sc.MinWaitingTime)}
	for _, sh := range sc.Requests {
		name, cnt, sleep, err := config.ParseShootName(sh)
		if err != nil {
			return nil, fmt.Errorf("failed to parse shoot %s: %w", sh, err)
		}
		if name == "sleep" {
			result.Calls[len(result.Calls)-1].Sleep += time.Millisecond * time.Duration(cnt)
			continue
		}
		req, ok := reqs[name]
		if !ok {
			return nil, fmt.Errorf("request %s not found", name)
		}
		r := convertConfigToStep(req, iter)
		if sleep > 0 {
			r.Sleep += time.Millisecond * time.Duration(sleep)
		}
		for i := 0; i < cnt; i++ {
			result.Calls = append(result.Calls, r)
		}
	}

	return result, nil
}

func convertConfigToStep(req config.CallConfig, iter mp.Iterator) gun.Call {
	postprocessors := make([]gun.Postprocessor, len(req.Postprocessors))
	copy(postprocessors, req.Postprocessors)
	preprocessors := make([]gun.Preprocessor, len(req.Preprocessors))
	for i := range req.Preprocessors {
		preprocessors[i] = req.Preprocessors[i]
		if p, ok := preprocessors[i].(IteratorIniter); ok {
			p.InitIterator(iter)
		}
	}
	result := gun.Call{
		Name:           req.Name,
		Preprocessors:  preprocessors,
		Postprocessors: postprocessors,
		Tag:            req.Tag,
		Call:           req.Call,
		Metadata:       req.Metadata,
		Payload:        []byte(req.Payload),
	}

	return result
}
