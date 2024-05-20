package config

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"
	grpcgun "github.com/yandex/pandora/components/guns/grpc/scenario"
	httpscenario "github.com/yandex/pandora/components/guns/http_scenario"
	"github.com/yandex/pandora/components/providers/scenario/http/preprocessor"
	"github.com/yandex/pandora/components/providers/scenario/vs"
)

// AmmoConfig is a config for dynamic converting from map[string]interface{}
type AmmoConfig struct {
	Locals          map[string]any
	VariableSources []vs.VariableSource `config:"variable_sources"`
	Requests        []RequestConfig
	Calls           []CallConfig
	Scenarios       []ScenarioConfig
}

// ScenarioConfig is a config for dynamic converting from map[string]interface{}
type ScenarioConfig struct {
	Name           string
	Weight         int64
	MinWaitingTime int64 `config:"min_waiting_time"`
	Requests       []string
}

// RequestConfig is a config for dynamic converting from map[string]interface{}
type RequestConfig struct {
	Name           string
	Method         string
	Headers        map[string]string
	Tag            string
	Body           *string
	URI            string
	Preprocessor   *preprocessor.Preprocessor
	Postprocessors []httpscenario.Postprocessor
	Templater      httpscenario.Templater
}

type CallConfig struct {
	Name           string
	Tag            string
	Call           string
	Payload        string
	Metadata       map[string]string
	Preprocessors  []grpcgun.Preprocessor
	Postprocessors []grpcgun.Postprocessor
}

func ReadAmmoConfig(fs afero.Fs, fileName string) (ammoCfg *AmmoConfig, err error) {
	const op = "scenario.ReadAmmoConfig"

	if fileName == "" {
		return nil, fmt.Errorf("scenario provider config should contain non-empty 'file' field")
	}
	file, openErr := fs.Open(fileName)
	if openErr != nil {
		return nil, fmt.Errorf("%s %w", op, openErr)
	}
	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			if err != nil {
				err = fmt.Errorf("%s multiple errors faced: %w, with close err: %s", op, err, closeErr)
			} else {
				err = fmt.Errorf("%s, %w", op, closeErr)
			}
		}
	}()
	stat, statErr := file.Stat()
	if statErr != nil {
		err = fmt.Errorf("%s file.Stat() %w", op, err)
		return
	}
	lowerName := strings.ToLower(stat.Name())
	switch {
	case strings.HasSuffix(lowerName, ".hcl"):
		ammoHcl, parseErr := ParseHCLFile(file)
		if parseErr != nil {
			err = fmt.Errorf("%s ParseHCLFile %w", op, parseErr)
			return
		}
		ammoCfg, err = ConvertHCLToAmmo(ammoHcl)
	case strings.HasSuffix(lowerName, ".yaml") || strings.HasPrefix(lowerName, ".yml"):
		ammoCfg, err = ParseAmmoConfig(file)
	default:
		err = fmt.Errorf("%s file extension should be .yaml or .yml", op)
		return
	}
	if err != nil {
		err = fmt.Errorf("%s ParseAmmoConfig %w", op, err)
		return
	}

	return ammoCfg, nil
}
