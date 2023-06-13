package raw

import (
	"errors"
	"net/http"
	"net/url"
	"testing"
	"testing/iotest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type DecoderHeaderWant struct {
	reqSize int
	tag     string
	err     error
}

func TestDecodeHeader(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  DecoderHeaderWant
	}{
		{
			name:  "should parse header with tag",
			input: "123 tag",
			want:  DecoderHeaderWant{123, "tag", nil},
		},
		{
			name:  "should parse header without tag",
			input: "123",
			want:  DecoderHeaderWant{123, "", nil},
		},
	}
	var ans DecoderHeaderWant
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ans.reqSize, ans.tag, ans.err = DecodeHeader(tt.input)
			assert.Equal(tt.want, ans)
		})
	}
}

type DecoderRequestWant struct {
	req  *http.Request
	err  error
	body []byte
}

func TestDecodeRequest(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  DecoderRequestWant
	}{
		{
			name: "should parse GET request",
			input: []byte("GET /some/path HTTP/1.0\r\n" +
				"Host: foo.com\r\n" +
				"Connection: close\r\n\r\n"),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "GET",
					URL:        MustURL(t, "/some/path"),
					Proto:      "HTTP/1.0",
					ProtoMajor: 1,
					ProtoMinor: 0,
					Host:       "foo.com",
					Header:     http.Header{"Connection": []string{"close"}},
					Body:       http.NoBody,
					Close:      true, // FIXME: BUG: should we say to close connection?
				}, nil, nil,
			},
		},
		{
			name: "should parse POST request with body",
			input: []byte("POST /some/path HTTP/1.1\r\n" +
				"Host: foo.com\r\n" +
				"Transfer-Encoding: chunked\r\n" +
				"Foo: bar\r\n" +
				"Content-Length: 9999\r\n\r\n" + // to be removed.
				"3\r\nfoo\r\n" +
				"3\r\nbar\r\n" +
				"0\r\n" +
				"\r\n"),
			want: DecoderRequestWant{
				&http.Request{
					Method:           "POST",
					URL:              MustURL(t, "/some/path"),
					Proto:            "HTTP/1.1",
					ProtoMajor:       1,
					ProtoMinor:       1,
					Host:             "foo.com",
					Header:           http.Header{"Foo": []string{"bar"}},
					Body:             nil,
					ContentLength:    -1,
					TransferEncoding: []string{"chunked"},
				}, nil, []byte("foobar"),
			},
		},
		{
			name: "should return error on bad urls",
			input: []byte("GET ../../../../etc/passwd HTTP/1.1\r\n" +
				"Host: foo.com\r\n" +
				"Content-Length: 0\r\n" +
				"\r\n"),
			want: DecoderRequestWant{
				nil, &url.Error{
					Op:  "parse",
					URL: "../../../../etc/passwd",
					Err: errors.New("invalid URI for request"),
				},
				nil,
			},
		},
		{
			name: "should replace header Host for URL if specified",
			input: []byte("GET /etc/passwd HTTP/1.1\r\n" +
				"Host: hostname.tld\r\n" +
				"Content-Length: 0\r\n" +
				"\r\n"),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "GET",
					URL:        MustURL(t, "/etc/passwd"),
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Host:       "hostname.tld",
					Header:     http.Header{"Content-Length": []string{"0"}},
					Body:       http.NoBody,
				}, nil, nil,
			},
		},
	}
	var ans DecoderRequestWant
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			ans.req, ans.err = DecodeRequest(tt.input)
			if tt.want.body != nil {
				assert.NotNil(ans.req)
				assert.NoError(iotest.TestReader(ans.req.Body, tt.want.body))
				ans.req.Body = nil
				tt.want.body = nil
			}
			assert.Equal(tt.want, ans)
		})
	}
}

func MustURL(t *testing.T, rawURL string) *url.URL {
	url, err := url.Parse(rawURL)
	require.NoError(t, err)
	return url
}
