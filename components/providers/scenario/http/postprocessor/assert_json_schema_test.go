package postprocessor

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestAssertJsonSchema_Process(t *testing.T) {
	type fields struct {
		Source string
		Schema string
	}

	type args struct {
		resp *http.Response
		body io.Reader
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "Validate response from file",
			fields: fields{
				Source: "file",
				Schema: "testdata/test_schema.json",
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil},
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.NoError,
		},
		{
			name: "Validate response from json-string",
			fields: fields{
				Source: "json_string",
				Schema: "{\"type\": \"object\"}",
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil},
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid Body",
			fields: fields{
				Source: "file",
				Schema: "testdata/test_schema.json",
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil},
				body: nil, // for this postprocessor we don't care about response, we need only body for validate
			},
			wantErr: assert.Error,
		},
		{
			name: "Invalid schema source",
			fields: fields{
				Source: "test_source",
				Schema: "testdata/test_schema.json",
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil},
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
		{
			name: "Invalid schema file path",
			fields: fields{
				Source: "file",
				Schema: "not_existing_schema.json",
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil},
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
		{
			name: "Invalid schema url path",
			fields: fields{
				Source: "url",
				Schema: "https://example.com",
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil},
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
		{
			name: "Invalid schema json-string",
			fields: fields{
				Source: "json_string",
				Schema: "",
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil},
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
		{
			name: "Schema validation error",
			fields: fields{
				Source: "json_string",
				Schema: "{\"type\": \"array\"}",
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil},
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AssertJsonSchema{
				Schema: tt.fields.Schema,
				Source: tt.fields.Source,
			}

			process, err := a.Process(tt.args.resp, tt.args.body)
			assert.Nil(t, process)
			tt.wantErr(t, err, fmt.Sprintf("Process(%v, %v)", tt.args.resp, tt.args.body))
		})
	}
}

func TestAssertUrlJsonSchema_Process(t *testing.T) {
	t.Run("Validate response with url schema", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(serveSchemaHandler))

		a := AssertJsonSchema{
			Schema: server.URL + "/?file=test_schema.json",
			Source: "url",
		}

		resp := &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}
		body := bytes.NewReader([]byte(`{"message": "Hello, World!"}`))

		process, err := a.Process(resp, body)

		server.Close()

		assert.Nil(t, process)
		assert.NoError(t, err)
	})
}

func serveSchemaHandler(w http.ResponseWriter, r *http.Request) {
	schemaName := r.URL.Query().Get("file")
	if schemaName == "" {
		http.Error(w, "Missing file name", http.StatusBadRequest)
		return
	}

	schemaPath := "testdata/" + schemaName

	data, err := os.ReadFile(schemaPath)
	if err != nil {
		http.Error(w, "Failed to load schema: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(data)
	if err != nil {
		log.Fatal(err)
	}
}
