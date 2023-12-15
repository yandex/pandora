package httpscenario

import (
	"fmt"
	"io"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/providers/http_scenario/postprocessor"
	"github.com/yandex/pandora/lib/str"
	"gopkg.in/yaml.v2"
)

type AmmoHCL struct {
	VariableSources []SourceHCL   `hcl:"variable_source,block" config:"variable_sources" yaml:"variable_sources"`
	Requests        []RequestHCL  `hcl:"request,block"`
	Scenarios       []ScenarioHCL `hcl:"scenario,block"`
}

type SourceHCL struct {
	Name            string             `hcl:"name,label"`
	Type            string             `hcl:"type,label"`
	File            *string            `hcl:"file" yaml:"file,omitempty"`
	Fields          *[]string          `hcl:"fields" yaml:"fields,omitempty"`
	IgnoreFirstLine *bool              `hcl:"ignore_first_line" yaml:"ignore_first_line,omitempty"`
	Delimiter       *string            `hcl:"delimiter" yaml:"delimiter,omitempty"`
	Variables       *map[string]string `hcl:"variables" yaml:"variables,omitempty"`
}

type RequestHCL struct {
	Name           string             `hcl:"name,label"`
	Method         string             `hcl:"method"`
	URI            string             `hcl:"uri"`
	Headers        map[string]string  `hcl:"headers" yaml:"headers,omitempty"`
	Tag            *string            `hcl:"tag" yaml:"tag,omitempty"`
	Body           *string            `hcl:"body" yaml:"body,omitempty"`
	Preprocessor   *PreprocessorHCL   `hcl:"preprocessor,block" yaml:"preprocessor,omitempty"`
	Postprocessors []PostprocessorHCL `hcl:"postprocessor,block" yaml:"postprocessors,omitempty"`
	Templater      *TemplaterHCL      `hcl:"templater,block" yaml:"templater,omitempty"`
}

type ScenarioHCL struct {
	Name           string   `hcl:"name,label"`
	Weight         *int64   `hcl:"weight" yaml:"weight,omitempty"`
	MinWaitingTime *int64   `hcl:"min_waiting_time" config:"min_waiting_time" yaml:"min_waiting_time,omitempty"`
	Requests       []string `hcl:"requests" yaml:"requests"`
}

type AssertSizeHCL struct {
	Val *int    `hcl:"val"`
	Op  *string `hcl:"op"`
}

type PostprocessorHCL struct {
	Type       string             `hcl:"type,label"`
	Mapping    *map[string]string `hcl:"mapping" yaml:"mapping,omitempty"`
	Headers    *map[string]string `hcl:"headers" yaml:"headers,omitempty"`
	Body       *[]string          `hcl:"body" yaml:"body,omitempty"`
	StatusCode *int               `hcl:"status_code" yaml:"status_code,omitempty"`
	Size       *AssertSizeHCL     `hcl:"size,block" yaml:"size,omitempty"`
}

type PreprocessorHCL struct {
	Mapping map[string]string `hcl:"mapping" yaml:"mapping,omitempty"`
}

type TemplaterHCL struct {
	Type string `hcl:"type" yaml:"type"`
}

func ParseHCLFile(file afero.File) (AmmoHCL, error) {
	const op = "hcl.ParseHCLFile"

	var config AmmoHCL
	bytes, err := io.ReadAll(file)
	if err != nil {
		return AmmoHCL{}, fmt.Errorf("%s, io.ReadAll, %w", op, err)
	}
	err = hclsimple.Decode(file.Name(), bytes, nil, &config)
	if err != nil {
		return AmmoHCL{}, fmt.Errorf("%s, hclsimple.Decode, %w", op, err)
	}
	return config, nil
}

func ConvertHCLToAmmo(ammo AmmoHCL) (AmmoConfig, error) {
	const op = "scenario.ConvertHCLToAmmo"
	bytes, err := yaml.Marshal(ammo)
	if err != nil {
		return AmmoConfig{}, fmt.Errorf("%s, cant yaml.Marshal: %w", op, err)
	}
	cfg, err := decodeMap(bytes)
	if err != nil {
		return AmmoConfig{}, fmt.Errorf("%s, decodeMap, %w", op, err)
	}
	return cfg, nil
}

func ConvertAmmoToHCL(ammo AmmoConfig) (AmmoHCL, error) {
	const op = "scenario.ConvertHCLToAmmo"

	var sources []SourceHCL
	if len(ammo.VariableSources) > 0 {
		sources = make([]SourceHCL, len(ammo.VariableSources))
		for i, s := range ammo.VariableSources {
			switch val := s.(type) {
			case *VariableSourceVariables:
				var variables map[string]string
				if val.Variables != nil {
					variables = make(map[string]string, len(val.Variables))
					for k, va := range val.Variables {
						variables[k] = str.FormatString(va)
					}
				}
				v := SourceHCL{
					Type:      "variables",
					Name:      val.Name,
					Variables: &variables,
				}
				sources[i] = v
			case *VariableSourceJSON:
				file := val.File
				v := SourceHCL{
					Type: "file/json",
					Name: val.Name,
					File: &file,
				}
				sources[i] = v
			case *VariableSourceCsv:
				var fields *[]string
				if val.Fields != nil {
					f := val.Fields
					fields = &f
				}
				ignoreFirstLine := val.IgnoreFirstLine
				delimiter := val.Delimiter
				file := val.File
				v := SourceHCL{
					Type:            "file/csv",
					Name:            val.Name,
					File:            &file,
					Fields:          fields,
					IgnoreFirstLine: &ignoreFirstLine,
					Delimiter:       &delimiter,
				}
				sources[i] = v
			default:
				return AmmoHCL{}, fmt.Errorf("%s variable source type %T not supported", op, val)
			}
		}

	}
	var requests []RequestHCL
	if len(ammo.Requests) > 0 {
		requests = make([]RequestHCL, len(ammo.Requests))
		for i, r := range ammo.Requests {
			var postprocessors []PostprocessorHCL
			if len(r.Postprocessors) > 0 {
				postprocessors = make([]PostprocessorHCL, len(r.Postprocessors))
				for j, p := range r.Postprocessors {
					switch val := p.(type) {
					case *postprocessor.VarHeaderPostprocessor:
						postprocessors[j] = PostprocessorHCL{
							Type:    "var/header",
							Mapping: &val.Mapping,
						}
					case *postprocessor.VarXpathPostprocessor:
						postprocessors[j] = PostprocessorHCL{
							Type:    "var/xpath",
							Mapping: &val.Mapping,
						}
					case *postprocessor.VarJsonpathPostprocessor:
						postprocessors[j] = PostprocessorHCL{
							Type:    "var/jsonpath",
							Mapping: &val.Mapping,
						}
					case *postprocessor.AssertResponse:
						postprocessors[j] = PostprocessorHCL{
							Type:       "assert/response",
							Headers:    &val.Headers,
							Body:       &val.Body,
							StatusCode: &val.StatusCode,
						}
						if val.Size != nil {
							postprocessors[j].Size = &AssertSizeHCL{
								Val: &val.Size.Val,
								Op:  &val.Size.Op,
							}
						}
						if e := val.Validate(); e != nil {
							return AmmoHCL{}, fmt.Errorf("%s postprocessor assert/response validation failed: %w", op, e)
						}
					default:
						return AmmoHCL{}, fmt.Errorf("%s postprocessor type %T not supported", op, val)
					}
				}
			}

			req := RequestHCL{
				Name:           r.Name,
				URI:            r.URI,
				Method:         r.Method,
				Headers:        r.Headers,
				Body:           r.Body,
				Postprocessors: postprocessors,
			}
			if r.Preprocessor.Mapping != nil {
				req.Preprocessor = &PreprocessorHCL{Mapping: r.Preprocessor.Mapping}
			}
			tag := r.Tag
			if tag != "" {
				req.Tag = &tag
			}
			templater := "text"
			_, ok := r.Templater.(*HTMLTemplater)
			if ok {
				templater = "html"
			}
			req.Templater = &TemplaterHCL{Type: templater}

			requests[i] = req
		}
	}
	var scenarios []ScenarioHCL
	if len(ammo.Scenarios) > 0 {
		scenarios = make([]ScenarioHCL, len(ammo.Scenarios))
		for i, s := range ammo.Scenarios {
			weight := s.Weight
			minWaitingTime := s.MinWaitingTime
			scenarios[i] = ScenarioHCL{
				Name:           s.Name,
				Requests:       s.Requests,
				Weight:         &weight,
				MinWaitingTime: &minWaitingTime,
			}
		}
	}

	result := AmmoHCL{
		VariableSources: sources,
		Requests:        requests,
		Scenarios:       scenarios,
	}

	return result, nil
}
