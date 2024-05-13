package config

import (
	"fmt"
	"io"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/spf13/afero"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
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

	bytes, err := io.ReadAll(file)
	if err != nil {
		return AmmoHCL{}, fmt.Errorf("%s, io.ReadAll, %w", op, err)
	}

	parser := hclparse.NewParser()
	f, diag := parser.ParseHCL(bytes, file.Name())
	if diag.HasErrors() {
		return AmmoHCL{}, diag
	}

	localsBodyContent, remainingBody, diag := f.Body.PartialContent(localsSchema())
	// diag still may have errors, because PartialContent doesn't know about Functions and self-references to locals
	if localsBodyContent == nil || remainingBody == nil {
		return AmmoHCL{}, diag
	}

	hclContext, diag := decodeLocals(localsBodyContent)
	if diag.HasErrors() {
		return AmmoHCL{}, diag
	}

	var config AmmoHCL
	diag = gohcl.DecodeBody(remainingBody, hclContext, &config)
	if diag.HasErrors() {
		return AmmoHCL{}, diag
	}
	return config, nil
}

func decodeLocals(localsBodyContent *hcl.BodyContent) (*hcl.EvalContext, hcl.Diagnostics) {
	vars := map[string]cty.Value{}
	hclContext := buildHclContext(vars)
	for _, block := range localsBodyContent.Blocks {
		if block == nil {
			continue
		}
		if block.Type == "locals" {
			newVars, err := decodeLocalBlock(block, hclContext)
			if err != nil {
				return nil, err
			}
			hclContext = buildHclContext(mergeMaps(vars, newVars))
		}
	}
	return hclContext, nil
}

func mergeMaps[K comparable, V any](to, from map[K]V) map[K]V {
	for k, v := range from {
		to[k] = v
	}
	return to
}

func decodeLocalBlock(localsBlock *hcl.Block, hclContext *hcl.EvalContext) (map[string]cty.Value, hcl.Diagnostics) {
	attrs, err := localsBlock.Body.JustAttributes()
	if err != nil {
		return nil, err
	}

	vars := map[string]cty.Value{}
	for name, attr := range attrs {
		val, err := attr.Expr.Value(hclContext)
		if err != nil {
			return nil, err
		}
		vars[name] = val
	}

	return vars, nil
}

func localsSchema() *hcl.BodySchema {
	return &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{
				Type:       "locals",
				LabelNames: []string{},
			},
		},
	}
}

func buildHclContext(vars map[string]cty.Value) *hcl.EvalContext {
	return &hcl.EvalContext{
		Variables: map[string]cty.Value{
			"local": cty.ObjectVal(vars),
		},
		Functions: map[string]function.Function{
			// collection functions
			"coalesce":     stdlib.CoalesceFunc,
			"coalescelist": stdlib.CoalesceListFunc,
			"compact":      stdlib.CompactFunc,
			"concat":       stdlib.ConcatFunc,
			"distinct":     stdlib.DistinctFunc,
			"element":      stdlib.ElementFunc,
			"flatten":      stdlib.FlattenFunc,
			"index":        stdlib.IndexFunc,
			"keys":         stdlib.KeysFunc,
			"lookup":       stdlib.LookupFunc,
			"merge":        stdlib.MergeFunc,
			"reverse":      stdlib.ReverseListFunc,
			"slice":        stdlib.SliceFunc,
			"sort":         stdlib.SortFunc,
			"split":        stdlib.SplitFunc,
			"values":       stdlib.ValuesFunc,
			"zipmap":       stdlib.ZipmapFunc,
		},
	}
}
