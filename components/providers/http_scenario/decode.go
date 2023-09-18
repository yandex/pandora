package httpscenario

import (
	"fmt"
	"io"
	"log"
	"strconv"
	"time"

	httpscenario "github.com/yandex/pandora/components/guns/http_scenario"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/lib/math"
	"github.com/yandex/pandora/lib/mp"
	"github.com/yandex/pandora/lib/str"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

func ParseAmmoConfig(file io.Reader) (AmmoConfig, error) {
	var ammoCfg AmmoConfig
	const op = "scenario/decoder.ParseAmmoConfig"
	data := make(map[string]any)
	bytes, err := io.ReadAll(file)
	if err != nil {
		return ammoCfg, fmt.Errorf("%s, io.ReadAll, %w", op, err)
	}
	err = yaml.Unmarshal(bytes, &data)
	if err != nil {
		return ammoCfg, fmt.Errorf("%s, yaml.Unmarshal, %w", op, err)
	}
	err = config.DecodeAndValidate(data, &ammoCfg)
	if err != nil {
		log.Fatal("Config decode failed", zap.Error(err))
	}
	return ammoCfg, nil
}

func decodeAmmo(cfg AmmoConfig, storage SourceStorage) ([]*Ammo, error) {
	reqRegistry := make(map[string]RequestConfig, len(cfg.Requests))

	for _, req := range cfg.Requests {
		reqRegistry[req.Name] = req
	}

	scenarioRegistry := map[string]ScenarioConfig{}
	for _, sc := range cfg.Scenarios {
		scenarioRegistry[sc.Name] = sc
	}

	names, size := spreadNames(cfg.Scenarios)
	result := make([]*Ammo, 0, size)
	for _, sc := range cfg.Scenarios {
		a, err := convertScenarioToAmmo(sc, reqRegistry)
		a.variableStorage = &storage
		if err != nil {
			return nil, fmt.Errorf("failed to convert scenario %s: %w", sc.Name, err)
		}
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

func convertScenarioToAmmo(sc ScenarioConfig, reqs map[string]RequestConfig) (*Ammo, error) {
	iter := mp.NewNextIterator(time.Now().UnixNano())
	result := &Ammo{name: sc.Name, minWaitingTime: time.Millisecond * time.Duration(sc.MinWaitingTime)}
	for _, sh := range sc.Requests {
		name, cnt, sleep, err := parseShootName(sh)
		if err != nil {
			return nil, fmt.Errorf("failed to parse shoot %s: %w", sh, err)
		}
		if name == "sleep" {
			result.Requests[len(result.Requests)-1].sleep += time.Millisecond * time.Duration(cnt)
			continue
		}
		req, ok := reqs[name]
		if !ok {
			return nil, fmt.Errorf("request %s not found", name)
		}
		r := convertConfigToRequest(req, iter)
		if sleep > 0 {
			r.sleep += time.Millisecond * time.Duration(sleep)
		}
		for i := 0; i < cnt; i++ {
			result.Requests = append(result.Requests, r)
		}
	}

	return result, nil
}

func convertConfigToRequest(req RequestConfig, iter mp.Iterator) Request {
	postprocessors := make([]httpscenario.Postprocessor, len(req.Postprocessors))
	for i := range req.Postprocessors {
		postprocessors[i] = req.Postprocessors[i].(httpscenario.Postprocessor)
	}
	templater := req.Templater
	if templater == nil {
		templater = NewTextTemplater()
	}
	result := Request{
		method:         req.Method,
		headers:        req.Headers,
		tag:            req.Tag,
		body:           req.Body,
		name:           req.Name,
		uri:            req.URI,
		preprocessor:   req.Preprocessor,
		postprocessors: postprocessors,
		templater:      templater,
	}
	result.preprocessor.iterator = iter

	return result
}

func parseShootName(shoot string) (string, int, int, error) {
	name, args, err := str.ParseStringFunc(shoot)
	if err != nil {
		return "", 0, 0, err
	}
	cnt := 1
	if len(args) > 0 && args[0] != "" {
		cnt, err = strconv.Atoi(args[0])
		if err != nil {
			return "", 0, 0, fmt.Errorf("failed to parse count: %w", err)
		}
	}
	sleep := 0
	if len(args) > 1 && args[1] != "" {
		sleep, err = strconv.Atoi(args[1])
		if err != nil {
			return "", 0, 0, fmt.Errorf("failed to parse count: %w", err)
		}
	}
	return name, cnt, sleep, nil
}

func spreadNames(input []ScenarioConfig) (map[string]int, int) {
	if len(input) == 0 {
		return nil, 0
	}
	if len(input) == 1 {
		return map[string]int{input[0].Name: 1}, 1
	}

	scenarioRegistry := map[string]ScenarioConfig{}
	weights := make([]int64, len(input))
	for i := range input {
		scenarioRegistry[input[i].Name] = input[i]
		if input[i].Weight == 0 {
			input[i].Weight = 1
		}
		weights[i] = input[i].Weight
	}

	div := math.GCDM(weights...)
	names := make(map[string]int)
	total := 0
	for _, sc := range input {
		cnt := int(sc.Weight / div)
		total += cnt
		names[sc.Name] = cnt
	}
	return names, total
}
