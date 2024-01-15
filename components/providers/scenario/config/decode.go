package config

import (
	"fmt"
	"io"
	"strconv"

	"github.com/yandex/pandora/components/providers/scenario/vs"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/lib/math"
	"github.com/yandex/pandora/lib/str"
	"gopkg.in/yaml.v2"
)

func ParseAmmoConfig(file io.Reader) (*AmmoConfig, error) {
	const op = "scenario/decoder.ParseAmmoConfig"
	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("%s, io.ReadAll, %w", op, err)
	}
	cfg, err := DecodeMap(bytes)
	if err != nil {
		return nil, fmt.Errorf("%s, decodeMap, %w", op, err)
	}
	return cfg, nil
}

func ConvertHCLToAmmo(ammo AmmoHCL) (*AmmoConfig, error) {
	const op = "scenario.ConvertHCLToAmmo"
	bytes, err := yaml.Marshal(ammo)
	if err != nil {
		return nil, fmt.Errorf("%s, cant yaml.Marshal: %w", op, err)
	}
	cfg, err := DecodeMap(bytes)
	if err != nil {
		return nil, fmt.Errorf("%s, decodeMap, %w", op, err)
	}
	return cfg, nil
}

func DecodeMap(bytes []byte) (*AmmoConfig, error) {
	const op = "scenario/decoder.decodeMap"

	var ammoCfg AmmoConfig

	data := make(map[string]any)
	err := yaml.Unmarshal(bytes, &data)
	if err != nil {
		return nil, fmt.Errorf("%s, yaml.Unmarshal, %w", op, err)
	}
	err = config.DecodeAndValidate(data, &ammoCfg)
	if err != nil {
		return nil, fmt.Errorf("%s, config.DecodeAndValidate, %w", op, err)
	}
	return &ammoCfg, nil
}

func ExtractVariableStorage(cfg *AmmoConfig) (*vs.SourceStorage, error) {
	storage := vs.NewVariableStorage()
	for _, source := range cfg.VariableSources {
		err := source.Init()
		if err != nil {
			return storage, err
		}
		storage.AddSource(source.GetName(), source.GetVariables())
	}
	return storage, nil
}

func ParseShootName(shoot string) (string, int, int, error) {
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

func SpreadNames(input []ScenarioConfig) (map[string]int, int) {
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
