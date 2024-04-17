package preprocessor

import (
	"errors"
	"fmt"

	"github.com/yandex/pandora/components/guns/grpc/scenario"
	"github.com/yandex/pandora/components/providers/scenario/templater"
	"github.com/yandex/pandora/lib/mp"
)

type PreprocessorConfig struct {
	Mapping map[string]string
}

type PreparePreprocessor struct {
	Mapping  map[string]string
	iterator mp.Iterator
}

func (p *PreparePreprocessor) InitIterator(iterator mp.Iterator) {
	p.iterator = iterator
}

func (p *PreparePreprocessor) Process(_ *scenario.Call, templateVars map[string]any) (map[string]any, error) {
	if templateVars == nil {
		return nil, errors.New("templateVars must not be nil")
	}
	result := make(map[string]any, len(p.Mapping))
	var (
		val any
		err error
	)
	for k, v := range p.Mapping {
		fun, args := templater.ParseFunc(v)
		if fun != nil {
			val, err = templater.ExecTemplateFuncWithVariables(fun, args, templateVars, p.iterator)
		} else {
			val, err = mp.GetMapValue(templateVars, v, p.iterator)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get value for %s: %w", k, err)
		}
		result[k] = val
	}
	return result, nil
}
