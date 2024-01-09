package config

import (
	"fmt"
	"io"

	"github.com/hashicorp/hcl/v2/hclsimple"
	"github.com/spf13/afero"
)

type AmmoHCL struct {
	VariableSources []SourceHCL   `hcl:"variable_source,block" config:"variable_sources" yaml:"variable_sources"`
	Requests        []RequestHCL  `hcl:"request,block"`
	Calls           []CallHCL     `hcl:"call,block"`
	Scenarios       []ScenarioHCL `hcl:"scenario,block"`
}

type ScenarioHCL struct {
	Name           string   `hcl:"name,label"`
	Weight         *int64   `hcl:"weight" yaml:"weight,omitempty"`
	MinWaitingTime *int64   `hcl:"min_waiting_time" config:"min_waiting_time" yaml:"min_waiting_time,omitempty"`
	Requests       []string `hcl:"requests" yaml:"requests"`
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
	Name           string                    `hcl:"name,label"`
	Method         string                    `hcl:"method"`
	URI            string                    `hcl:"uri"`
	Headers        map[string]string         `hcl:"headers" yaml:"headers,omitempty"`
	Tag            *string                   `hcl:"tag" yaml:"tag,omitempty"` //TODO: remove
	Body           *string                   `hcl:"body" yaml:"body,omitempty"`
	Preprocessor   *RequestPreprocessorHCL   `hcl:"preprocessor,block" yaml:"preprocessor,omitempty"`
	Postprocessors []RequestPostprocessorHCL `hcl:"postprocessor,block" yaml:"postprocessors,omitempty"`
	Templater      *TemplaterHCL             `hcl:"templater,block" yaml:"templater,omitempty"`
}

type TemplaterHCL struct {
	Type string `hcl:"type" yaml:"type"`
}

type AssertSizeHCL struct {
	Val *int    `hcl:"val"`
	Op  *string `hcl:"op"`
}

type RequestPostprocessorHCL struct {
	Type       string             `hcl:"type,label"`
	Mapping    *map[string]string `hcl:"mapping" yaml:"mapping,omitempty"`
	Headers    *map[string]string `hcl:"headers" yaml:"headers,omitempty"`
	Body       *[]string          `hcl:"body" yaml:"body,omitempty"`
	StatusCode *int               `hcl:"status_code" yaml:"status_code,omitempty"`
	Size       *AssertSizeHCL     `hcl:"size,block" yaml:"size,omitempty"`
}

type RequestPreprocessorHCL struct {
	//Type    string            `hcl:"type,label"`
	Mapping map[string]string `hcl:"mapping"`
}

type CallHCL struct {
	Name           string                 `hcl:"name,label"`
	Tag            *string                `hcl:"tag" yaml:"tag,omitempty"`
	Call           string                 `hcl:"call"`
	Metadata       *map[string]string     `hcl:"metadata" yaml:"metadata,omitempty"`
	Payload        string                 `hcl:"payload"`
	Preprocessor   []CallPreprocessorHCL  `hcl:"preprocessor,block" yaml:"preprocessors,omitempty"`
	Postprocessors []CallPostprocessorHCL `hcl:"postprocessor,block" yaml:"postprocessors,omitempty"`
}

type CallPostprocessorHCL struct {
	Type       string    `hcl:"type,label"`
	Payload    *[]string `hcl:"payload" yaml:"payload,omitempty"`
	StatusCode *int      `hcl:"status_code" yaml:"status_code,omitempty"`
}

type CallPreprocessorHCL struct {
	Type    string            `hcl:"type,label"`
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
