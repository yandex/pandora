package postprocessor

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAssertResponse_Process(t *testing.T) {
	type fields struct {
		Headers    map[string]string
		Body       []string
		StatusCode int
		Size       *AssertSize
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
			name: "Valid Response",
			fields: fields{
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []string{"Hello, World!"},
				StatusCode: http.StatusOK,
				Size:       &AssertSize{Val: 13, Op: ">"},
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}, // Set body to nil for this example
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid Header",
			fields: fields{
				Headers:    map[string]string{"Content-Type": "application/xml"},
				Body:       []string{"Hello, World!"},
				StatusCode: http.StatusOK,
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}, // Set body to nil for this example
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
		{
			name: "Invalid Body",
			fields: fields{
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []string{"Invalid Text"},
				StatusCode: http.StatusOK,
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}, // Set body to nil for this example
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
		{
			name: "Empty Body",
			fields: fields{
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []string{"Hello, World!"},
				StatusCode: http.StatusOK,
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}, // Set body to nil for this example
				body: nil,                                                                                                                     // Set body to nil for this example
			},
			wantErr: assert.Error,
		},
		{
			name: "Invalid StatusCode",
			fields: fields{
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []string{"Hello, World!"},
				StatusCode: http.StatusOK,
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}, // Set body to nil for this example
				body: nil,                                                                                                                           // Set body to nil for this example
			},
			wantErr: assert.Error,
		},
		{
			name: "Valid Size Assertion",
			fields: fields{
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []string{"Hello, World!"},
				StatusCode: http.StatusOK,
				Size:       &AssertSize{Val: 28, Op: "eq"},
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}, // Set body to nil for this example
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.NoError,
		},
		{
			name: "Invalid Size Assertion",
			fields: fields{
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []string{"Hello, World!"},
				StatusCode: http.StatusOK,
				Size:       &AssertSize{Val: 20, Op: "lt"},
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}, // Set body to nil for this example
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
		{
			name: "Invalid Size Assertion",
			fields: fields{
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []string{"Hello, World!"},
				StatusCode: http.StatusOK,
				Size:       &AssertSize{Val: 40, Op: "gt"},
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}, // Set body to nil for this example
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
		{
			name: "Unknown Size Assertion Operator",
			fields: fields{
				Headers:    map[string]string{"Content-Type": "application/json"},
				Body:       []string{"Hello, World!"},
				StatusCode: http.StatusOK,
				Size:       &AssertSize{Val: 13, Op: "unknown"},
			},
			args: args{
				resp: &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: nil}, // Set body to nil for this example
				body: bytes.NewReader([]byte(`{"message": "Hello, World!"}`)),
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := AssertResponse{
				Headers:    tt.fields.Headers,
				Body:       tt.fields.Body,
				StatusCode: tt.fields.StatusCode,
				Size:       tt.fields.Size,
			}
			process, err := a.Process(tt.args.resp, tt.args.body)
			assert.Nil(t, process)
			tt.wantErr(t, err, fmt.Sprintf("Process(%v, %v)", tt.args.resp, tt.args.body))
		})
	}
}
