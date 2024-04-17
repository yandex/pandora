package vs

import (
	"fmt"

	"github.com/yandex/pandora/components/providers/scenario/templater"
)

type VariableSourceVariables struct {
	Name      string
	Variables map[string]any
}

func (v *VariableSourceVariables) GetName() string {
	return v.Name
}

func (v *VariableSourceVariables) GetVariables() any {
	return v.Variables
}

func (v *VariableSourceVariables) Init() error {
	return v.recursiveCompute(v.Variables)
}

func (v *VariableSourceVariables) recursiveCompute(input map[string]any) error {
	var err error
	for key, val := range input {
		switch value := val.(type) {
		case string:
			input[key], err = v.execTemplateFunc(value)
			if err != nil {
				return fmt.Errorf("recursiveCompute for %s err: %w", key, err)
			}
		case map[string]any:
			err := v.recursiveCompute(value)
			if err != nil {
				return fmt.Errorf("recursiveCompute for %s err: %w", key, err)
			}
		case map[string]string:
			for k, vv := range value {
				value[k], err = v.execTemplateFunc(vv)
				if err != nil {
					return fmt.Errorf("recursiveCompute for %s err: %w", key, err)
				}
			}
			input[key] = value
		case []string:
			for i, vv := range value {
				value[i], err = v.execTemplateFunc(vv)
				if err != nil {
					return fmt.Errorf("recursiveCompute for %s err: %w", key, err)
				}
			}
			input[key] = value
		}
	}
	return nil
}

func (v *VariableSourceVariables) execTemplateFunc(in string) (string, error) {
	fun, args := templater.ParseFunc(in)
	if fun == nil {
		return in, nil
	}
	value, err := templater.ExecTemplateFunc(fun, args)
	if err != nil {
		return "", err
	}
	return value, nil
}
