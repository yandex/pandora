package jsonline

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"testing/iotest"

	// . "github.com/onsi/ginkgo"
	// . "github.com/onsi/gomega"
	// . "github.com/onsi/gomega/gstruct"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/yandex/pandora/lib/testutil"
)

const testFile = "./ammo.jsonline"

// testData holds jsonline.data that contains in testFile
var testData = []data{
	{
		Host:    "example.com",
		Method:  "GET",
		URI:     "/00",
		Headers: map[string]string{"Accept": "*/*", "Accept-Encoding": "gzip, deflate", "User-Agent": "Pandora/0.0.1"},
	},
	{
		Host:    "ya.ru",
		Method:  "HEAD",
		URI:     "/01",
		Headers: map[string]string{"Accept": "*/*", "Accept-Encoding": "gzip, brotli", "User-Agent": "YaBro/0.1"},
		Tag:     "head",
	},
}

var testFs = newTestFs()

func newTestFs() afero.Fs {
	fs := afero.NewMemMapFs()
	file, err := fs.Create(testFile)
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(file)
	for _, d := range testData {
		err := encoder.Encode(d)
		if err != nil {
			panic(err)
		}
	}
	return afero.NewReadOnlyFs(fs)
}

type ToRequestWant struct {
	req *http.Request
	error
	body []byte
}

func TestToRequest(t *testing.T) {
	var tests = []struct {
		name  string
		input data
		want  ToRequestWant
	}{
		{
			name: "decoded well",
			input: data{
				Host:    "ya.ru",
				Method:  "GET",
				URI:     "/00",
				Headers: map[string]string{"A": "a", "B": "b"},
				Tag:     "tag",
			},
			want: ToRequestWant{
				req: &http.Request{
					Method:     "GET",
					URL:        testutil.Must(url.Parse("http://ya.ru/00")),
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Header:     http.Header{"A": []string{"a"}, "B": []string{"b"}},
					Body:       http.NoBody,
					Host:       "ya.ru",
				},
				error: nil,
			},
		},
	}
	var ans ToRequestWant
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ans.req, ans.error = tt.input.ToRequest()
			if tt.want.body != nil {
				assert.NotNil(ans.req)
				assert.NoError(iotest.TestReader(ans.req.Body, tt.want.body))
				ans.req.Body = nil
				tt.want.body = nil
			}
			ans.req.GetBody = nil
			tt.want.req = ans.req.WithContext(context.Background())
			assert.Equal(tt.want, ans)
		})
	}
}

type DecodeAmmoWant struct {
	req *http.Request
	tag string
	error
}

func TestDecodeAmmo(t *testing.T) {
	var tests = []struct {
		name  string
		input []byte
		want  DecodeAmmoWant
	}{}
	var ans DecodeAmmoWant
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ans.req, ans.tag, ans.error = DecodeAmmo(tt.input)
			assert.Equal(tt.want, ans)
		})
	}
}

func BenchmarkDecodeAmmo(b *testing.B) {
	jsonDoc, err := json.Marshal(testData[0])
	assert.NoError(b, err)
	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		_, _, _ = DecodeAmmo(jsonDoc)
	}
}
