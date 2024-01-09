package http

import (
	"fmt"
	"time"

	gun "github.com/yandex/pandora/components/guns/http_scenario"
	"github.com/yandex/pandora/components/providers/scenario/config"
	"github.com/yandex/pandora/components/providers/scenario/http/templater"
	"github.com/yandex/pandora/components/providers/scenario/vs"
	"github.com/yandex/pandora/lib/mp"
)

type IteratorIniter interface {
	InitIterator(iter mp.Iterator)
}

func decodeAmmo(cfg *config.AmmoConfig, storage *vs.SourceStorage) ([]*gun.Scenario, error) {
	reqRegistry := make(map[string]config.RequestConfig, len(cfg.Requests))

	for _, req := range cfg.Requests {
		reqRegistry[req.Name] = req
	}

	scenarioRegistry := map[string]config.ScenarioConfig{}
	for _, sc := range cfg.Scenarios {
		scenarioRegistry[sc.Name] = sc
	}

	names, size := config.SpreadNames(cfg.Scenarios)
	result := make([]*gun.Scenario, 0, size)
	for _, sc := range cfg.Scenarios {
		a, err := convertScenarioToAmmo(sc, reqRegistry)
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

func convertScenarioToAmmo(sc config.ScenarioConfig, reqs map[string]config.RequestConfig) (*gun.Scenario, error) {
	iter := mp.NewNextIterator(time.Now().UnixNano())
	result := &gun.Scenario{Name: sc.Name, MinWaitingTime: time.Millisecond * time.Duration(sc.MinWaitingTime)}
	for _, sh := range sc.Requests {
		name, cnt, sleep, err := config.ParseShootName(sh)
		if err != nil {
			return nil, fmt.Errorf("failed to parse shoot %s: %w", sh, err)
		}
		if name == "sleep" {
			result.Requests[len(result.Requests)-1].Sleep += time.Millisecond * time.Duration(cnt)
			continue
		}
		req, ok := reqs[name]
		if !ok {
			return nil, fmt.Errorf("request %s not found", name)
		}
		r := convertConfigToRequest(req, iter)
		if sleep > 0 {
			r.Sleep += time.Millisecond * time.Duration(sleep)
		}
		for i := 0; i < cnt; i++ {
			result.Requests = append(result.Requests, r)
		}
	}

	return result, nil
}

func convertConfigToRequest(req config.RequestConfig, iter mp.Iterator) gun.Request {
	templ := req.Templater
	if templ == nil {
		templ = templater.NewTextTemplater()
	}
	result := gun.Request{
		Method:         req.Method,
		Headers:        req.Headers,
		Tag:            req.Tag,
		Body:           req.Body,
		Name:           req.Name,
		URI:            req.URI,
		Preprocessor:   req.Preprocessor,
		Postprocessors: req.Postprocessors,
		Templater:      templ,
	}
	if p, ok := result.Preprocessor.(IteratorIniter); ok {
		p.InitIterator(iter)
	}

	return result
}
