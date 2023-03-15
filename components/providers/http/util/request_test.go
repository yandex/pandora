package util

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func BenchmarkGetKeyValue(b *testing.B) {
	var h = "[hello: world]"
	for i := 0; i < b.N; i++ {
		_, _, _ = DecodeHeader(h)
	}
}

type GetKeyValueWant struct {
	key, value string
	err        error
}

func TestGetKeyValue(t *testing.T) {
	var tests = []struct {
		name  string
		input string
		want  GetKeyValueWant
	}{
		{
			name:  "empty input",
			input: "",
			want: GetKeyValueWant{
				"",
				"",
				ErrHeaderFormat,
			},
		},
		{
			name:  "no colon",
			input: "[key value]",
			want: GetKeyValueWant{
				"key value",
				"",
				ErrHeaderFormat,
			},
		},
		{
			name:  "simple input",
			input: "[key: value]",
			want: GetKeyValueWant{
				"key",
				"value",
				nil,
			},
		},
		{
			name:  "empty output exected",
			input: "[:]",
			want: GetKeyValueWant{
				"",
				"",
				ErrEmptyKey,
			},
		},
	}
	var ans GetKeyValueWant
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ans.key, ans.value, ans.err = DecodeHeader(tt.input)
			assert.EqualValues(tt.want, ans)
		})
	}
}

type DecodeHTTPConfigHeadersWant struct {
	ans map[string][]string
	err error
}

func TestDecodeHTTPConfigHeaders(t *testing.T) {
	var tests = []struct {
		name  string
		input []string
		want  DecodeHTTPConfigHeadersWant
	}{
		{
			name: "decode http config headers",
			input: []string{
				"[Host: youhost.tld]",
				"[SomeHeader: somevalue]",
				"[soMeHeAdEr: secondvalue]",
			},
			want: DecodeHTTPConfigHeadersWant{
				map[string][]string{
					"Host":       {"youhost.tld"},
					"Someheader": {"somevalue", "secondvalue"},
				},
				nil,
			},
		},
		{
			name: "parse error",
			input: []string{
				"",
			},
			want: DecodeHTTPConfigHeadersWant{
				map[string][]string{},
				ErrHeaderFormat,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ans, err := DecodeHTTPConfigHeaders(tt.input)
			assert.EqualValues(tt.want.ans, ans)
			assert.EqualValues(tt.want.err, err)
		})
	}
}

type EnrichRequestWithHeadersInput struct {
	req     func() *http.Request
	headers http.Header
}

type EnrichRequestWithHeadersWant struct {
	Host string
	http.Header
}

func TestEnrichRequestWithHeaders(t *testing.T) {
	const origHost = "example.com"
	defaultReq, _ := http.NewRequest("GET", origHost, nil)

	var tests = []struct {
		name  string
		input EnrichRequestWithHeadersInput
		want  EnrichRequestWithHeadersWant
	}{
		{
			name: "add new http headers",
			input: EnrichRequestWithHeadersInput{
				req: func() (r *http.Request) {
					r = defaultReq.Clone(context.Background())
					r.Host = origHost
					r.Header.Set("SomeHeader", "oldvalue")
					return
				},
				headers: http.Header{
					"Host":         {"youhost.tld"},
					"Someheader":   {"newvalue"},
					"Secondheader": {"new_second_value"},
				},
			},
			want: EnrichRequestWithHeadersWant{
				Host: origHost,
				Header: http.Header{
					"Someheader":   []string{"oldvalue"},
					"Secondheader": []string{"new_second_value"},
				},
			},
		},
		{
			name: "add new host",
			input: EnrichRequestWithHeadersInput{
				req: func() (r *http.Request) {
					r = defaultReq.Clone(context.Background())
					return
				},
				headers: http.Header{
					"Host": {"youhost.tld"},
				},
			},
			want: EnrichRequestWithHeadersWant{
				Host:   "youhost.tld",
				Header: http.Header{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			req := tt.input.req()
			fmt.Println(req)
			EnrichRequestWithHeaders(req, tt.input.headers)
			fmt.Println(req, req.Host)
			assert.EqualValues(tt.want.Host, req.Host)
			assert.EqualValues(tt.want.Header, req.Header)
		})
	}
}
