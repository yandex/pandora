package decoders

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yandex/pandora/components/providers/http/config"
)

func Test_uriDecoder_readLine(t *testing.T) {
	tests := []struct {
		name                  string
		data                  string
		expectedReq           *http.Request
		expectedTag           string
		expectedErr           bool
		expectedCommonHeaders http.Header
	}{
		{
			name:        "Header line",
			data:        "[Content-Type: application/json]",
			expectedReq: nil,
			expectedTag: "",
			expectedErr: false,
			expectedCommonHeaders: http.Header{
				"Content-Type": []string{"application/json"},
				"User-Agent":   []string{"TestAgent"},
			},
		},
		{
			name: "Valid URI",
			data: "http://example.com/test",
			expectedReq: &http.Request{
				Method: "GET",
				Proto:  "HTTP/1.1",
				URL:    &url.URL{Scheme: "http", Host: "example.com", Path: "/test"},
				Header: http.Header{
					"User-Agent":    []string{"TestAgent"},
					"Authorization": []string{"Bearer xxx"},
				},
				Host:       "example.com",
				ProtoMajor: 1,
				ProtoMinor: 1,
			},
			expectedTag: "",
			expectedErr: false,
			expectedCommonHeaders: http.Header{
				"User-Agent": []string{"TestAgent"},
			},
		},
		{
			name: "URI with tag",
			data: "http://example.com/test tag\n",
			expectedReq: &http.Request{
				Method: "GET",
				Proto:  "HTTP/1.1",
				URL:    &url.URL{Scheme: "http", Host: "example.com", Path: "/test"},
				Header: http.Header{
					"User-Agent":    []string{"TestAgent"},
					"Authorization": []string{"Bearer xxx"},
				},
				Host:       "example.com",
				ProtoMajor: 1,
				ProtoMinor: 1,
			},
			expectedTag: "tag",
			expectedErr: false,
			expectedCommonHeaders: http.Header{
				"User-Agent": []string{"TestAgent"},
			},
		},
		{
			name:        "Invalid data",
			data:        "1http://foo.com tag",
			expectedReq: nil,
			expectedTag: "",
			expectedErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commonHeader := http.Header{"User-Agent": []string{"TestAgent"}}
			decodedConfigHeaders := http.Header{"Authorization": []string{"Bearer xxx"}}

			decoder := newURIDecoder(nil, config.Config{}, decodedConfigHeaders)
			req, tag, err := decoder.readLine(test.data, commonHeader)

			if test.expectedReq != nil {
				test.expectedReq = test.expectedReq.WithContext(context.Background())
			}

			if test.expectedErr {
				assert.Error(t, err)
				return
			}
			assert.Equal(t, test.expectedTag, tag)
			assert.Equal(t, test.expectedCommonHeaders, commonHeader)
			assert.Equal(t, test.expectedReq, req)
		})
	}
}

const uriInput = ` /0
[A:b]
/1
[Host : example.com]
[ C : d ]
/2
[A:]
[Host : other.net]

/3
/4 some tag`

func Test_uriDecoder_Scan(t *testing.T) {
	decoder := newURIDecoder(strings.NewReader(uriInput), config.Config{
		Limit: 10,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	tests := []struct {
		wantTag  string
		wantErr  bool
		wantBody string
	}{
		{
			wantTag:  "",
			wantErr:  false,
			wantBody: "GET /0 HTTP/1.1\r\nContent-Type: application/json\r\n\r\n",
		},
		{
			wantTag:  "",
			wantErr:  false,
			wantBody: "GET /1 HTTP/1.1\r\nA: b\r\nContent-Type: application/json\r\n\r\n",
		},
		{
			wantTag:  "",
			wantErr:  false,
			wantBody: "GET /2 HTTP/1.1\r\nHost: example.com\r\nA: b\r\nC: d\r\nContent-Type: application/json\r\n\r\n",
		},
		{
			wantTag:  "",
			wantErr:  false,
			wantBody: "GET /3 HTTP/1.1\r\nHost: other.net\r\nA: \r\nC: d\r\nContent-Type: application/json\r\n\r\n",
		},
		{
			wantTag:  "some tag",
			wantErr:  false,
			wantBody: "GET /4 HTTP/1.1\r\nHost: other.net\r\nA: \r\nC: d\r\nContent-Type: application/json\r\n\r\n",
		},
	}

	for j := 0; j < 2; j++ {
		for i, tt := range tests {
			req, tag, err := decoder.Scan(ctx)
			if tt.wantErr {
				assert.Error(t, err, "iteration %d-%d", j, i)
				continue
			} else {
				assert.NoError(t, err, "iteration %d-%d", j, i)
			}
			assert.Equal(t, tt.wantTag, tag, "iteration %d-%d", j, i)

			req.Close = false
			body, _ := httputil.DumpRequest(req, true)
			assert.Equal(t, tt.wantBody, string(body), "iteration %d-%d", j, i)
		}
	}

	_, _, err := decoder.Scan(ctx)
	assert.Equal(t, err, ErrAmmoLimit)
	assert.Equal(t, decoder.ammoNum, uint(len(tests)*2))
	assert.Equal(t, decoder.passNum, uint(1))
}
