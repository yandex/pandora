package postprocessor

import (
	"fmt"
	"github.com/xeipuuv/gojsonschema"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

type AssertJsonSchema struct {
	Source string `config:"source"`
	Schema string `config:"schema"`
}

func (a AssertJsonSchema) Process(_ *http.Response, body io.Reader) (map[string]any, error) {
	var schemaLoader gojsonschema.JSONLoader

	switch a.Source {
	case "file":
		absPath, err := filepath.Abs(a.Schema)

		if err != nil {
			return nil, fmt.Errorf("assert/json-schema fail to get schema file absolute path: %w", err)
		}

		filePath := "file:///" + absPath
		schemaLoader = gojsonschema.NewReferenceLoader(filePath)

	case "url":
		schemaLoader = gojsonschema.NewReferenceLoader(a.Schema)
	case "json_string":
		schemaLoader = gojsonschema.NewStringLoader(a.Schema)
	default:
		return nil, fmt.Errorf("unknown schema source %s", a.Source)
	}

	var b []byte
	var err error

	if body != nil {
		b, err = io.ReadAll(body)
		if err != nil {
			return nil, fmt.Errorf("assert/json-schema can't read body: %w", err)
		}
	}

	schema, err := gojsonschema.NewSchema(schemaLoader)

	if err != nil {
		return nil, fmt.Errorf("assert/json-schema can't create schema: %w", err)
	}

	respJson := gojsonschema.NewBytesLoader(b)
	result, err := schema.Validate(respJson)

	if err != nil {
		return nil, fmt.Errorf("assert/json-schema validate error: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}

		return nil, fmt.Errorf("assert/json-schema validation errors: %s", strings.Join(errors, ", "))
	}

	return nil, nil
}

func (a AssertJsonSchema) Validate() error {

	if a.Source != "file" && a.Source != "url" && a.Source != "json_string" {
		return fmt.Errorf("assert/json-schema unknown schema source %s", a.Source)
	}

	if a.Schema == "" {
		var invalidSource string

		switch a.Source {
		case "file":
			invalidSource = "file path"
		case "url":
			invalidSource = "url"
		case "json_string":
			invalidSource = "json string"
		}

		return fmt.Errorf("assert/json-schema invalid schema %s", invalidSource)
	}

	return nil
}
