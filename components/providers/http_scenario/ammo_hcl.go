package httpscenario

import (
	"fmt"
	"io"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/providers/http_scenario/postprocessor"
	"github.com/yandex/pandora/lib/str"
)

type AmmoHCL struct {
	VariableSources []SourceHCL   `hcl:"variable_source,block"`
	Requests        []RequestHCL  `hcl:"request,block"`
	Scenarios       []ScenarioHCL `hcl:"scenario,block"`
}

type SourceHCL struct {
	Name            string             `hcl:"name,label"`
	Type            string             `hcl:"type,label"`
	File            *string            `hcl:"file"`
	Fields          *[]string          `hcl:"fields"`
	IgnoreFirstLine *bool              `hcl:"ignore_first_line"`
	Delimiter       *string            `hcl:"delimiter"`
	Variables       *map[string]string `hcl:"variables"`
}

type RequestHCL struct {
	Name           string             `hcl:"name,label"`
	Method         string             `hcl:"method"`
	Headers        map[string]string  `hcl:"headers"`
	Tag            *string            `hcl:"tag"`
	Body           *string            `hcl:"body"`
	URI            string             `hcl:"uri"`
	Preprocessor   *PreprocessorHCL   `hcl:"preprocessor,block"`
	Postprocessors []PostprocessorHCL `hcl:"postprocessor,block"`
	Templater      *string            `hcl:"templater"`
}

type ScenarioHCL struct {
	Name           string   `hcl:"name,label"`
	Weight         *int64   `hcl:"weight"`
	MinWaitingTime *int64   `hcl:"min_waiting_time"`
	Requests       []string `hcl:"requests"`
}

type AssertSizeHCL struct {
	Val *int    `hcl:"val"`
	Op  *string `hcl:"op"`
}

type PostprocessorHCL struct {
	Type       string             `hcl:"type,label"`
	Mapping    *map[string]string `hcl:"mapping"`
	Headers    *map[string]string `hcl:"headers"`
	Body       *[]string          `hcl:"body"`
	StatusCode *int               `hcl:"status_code"`
	Size       *AssertSizeHCL     `hcl:"size,block"`
}

type PreprocessorHCL struct {
	Mapping map[string]string `hcl:"mapping"`
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

func ConvertHCLToAmmo(ammo AmmoHCL, fs afero.Fs) (AmmoConfig, error) {
	const op = "scenario.ConvertHCLToAmmo"

	var sources []VariableSource
	if len(ammo.VariableSources) > 0 {
		sources = make([]VariableSource, len(ammo.VariableSources))
		for i, s := range ammo.VariableSources {
			file := ""
			if s.File != nil {
				file = *s.File
			}
			switch s.Type {
			case "file/json":
				sources[i] = &VariableSourceJSON{
					Name: s.Name,
					File: file,
					fs:   fs,
				}
			case "file/csv":
				var fields []string
				if s.Fields != nil {
					fields = make([]string, len(*s.Fields))
					copy(fields, *s.Fields)
				}
				skipHeader := false
				if s.IgnoreFirstLine != nil {
					skipHeader = *s.IgnoreFirstLine
				}
				headerAsFields := ""
				if s.Delimiter != nil {
					headerAsFields = *s.Delimiter
				}
				sources[i] = &VariableSourceCsv{
					Name:            s.Name,
					File:            file,
					Fields:          fields,
					IgnoreFirstLine: skipHeader,
					Delimiter:       headerAsFields,
					fs:              fs,
				}
			default:
				return AmmoConfig{}, fmt.Errorf("%s, unknown variable source type: %s", op, s.Type)
			}
		}
	}

	var requests []RequestConfig
	if len(ammo.Requests) > 0 {
		requests = make([]RequestConfig, len(ammo.Requests))
		for i, r := range ammo.Requests {
			var postprocessors []postprocessor.Postprocessor
			if len(r.Postprocessors) > 0 {
				postprocessors = make([]postprocessor.Postprocessor, len(r.Postprocessors))
				for j, p := range r.Postprocessors {
					switch p.Type {
					case "var/header":
						postprocessors[j] = &postprocessor.VarHeaderPostprocessor{
							Mapping: *p.Mapping,
						}
					case "var/xpath":
						postprocessors[j] = &postprocessor.VarXpathPostprocessor{
							Mapping: *p.Mapping,
						}
					case "var/jsonpath":
						postprocessors[j] = &postprocessor.VarJsonpathPostprocessor{
							Mapping: *p.Mapping,
						}
					case "assert/response":
						postp := &postprocessor.AssertResponse{}
						if p.Headers != nil {
							postp.Headers = *p.Headers
						}
						if p.Body != nil {
							postp.Body = *p.Body
						}
						if p.StatusCode != nil {
							postp.StatusCode = *p.StatusCode
						}
						if p.Size != nil {
							postp.Size = &postprocessor.AssertSize{}
							if p.Size.Val != nil {
								postp.Size.Val = *p.Size.Val
							}
							if p.Size.Op != nil {
								postp.Size.Op = *p.Size.Op
							}
						}
						if err := postp.Validate(); err != nil {
							return AmmoConfig{}, fmt.Errorf("%s, invalid postprocessor.AssertResponse %w", op, err)
						}
						postprocessors[j] = postp
					default:
						return AmmoConfig{}, fmt.Errorf("%s, unknown postprocessor type: %s", op, p.Type)
					}
				}
			}
			templater := NewTextTemplater()
			if r.Templater != nil && *r.Templater == "html" {
				templater = NewHTMLTemplater()
			}
			tag := ""
			if r.Tag != nil {
				tag = *r.Tag
			}
			var variables map[string]string
			if r.Preprocessor != nil {
				variables = r.Preprocessor.Mapping
			}
			requests[i] = RequestConfig{
				Name:           r.Name,
				Method:         r.Method,
				Headers:        r.Headers,
				Tag:            tag,
				Body:           r.Body,
				URI:            r.URI,
				Preprocessor:   Preprocessor{Mapping: variables},
				Postprocessors: postprocessors,
				Templater:      templater,
			}
		}
	}

	var scenarios []ScenarioConfig
	if len(ammo.Scenarios) > 0 {
		scenarios = make([]ScenarioConfig, len(ammo.Scenarios))
		for i, s := range ammo.Scenarios {
			scenarios[i] = ScenarioConfig{
				Name:     s.Name,
				Requests: s.Requests,
			}
			if s.Weight != nil {
				scenarios[i].Weight = *s.Weight
			}
			if s.MinWaitingTime != nil {
				scenarios[i].MinWaitingTime = *s.MinWaitingTime
			}
		}
	}

	result := AmmoConfig{
		VariableSources: sources,
		Requests:        requests,
		Scenarios:       scenarios,
	}

	return result, nil
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
			req.Templater = &templater

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
