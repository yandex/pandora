package raw

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
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
			name: "should read body when Content-Length is missing (HTTP/1.0)",
			input: []byte("POST /some/path HTTP/1.0\r\n" +
				"Host: foo.com\r\n\r\n" +
				"hello"),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "POST",
					URL:        MustURL(t, "/some/path"),
					Proto:      "HTTP/1.0",
					ProtoMajor: 1,
					ProtoMinor: 0,
					Host:       "foo.com",
					Header:     http.Header{},
					Body:       nil,
					Close:      true,
				}, nil, []byte("hello"),
			},
		},
		{
			name: "should read body when Content-Length is missing (HTTP/1.1)",
			input: []byte("POST /some/path HTTP/1.1\r\n" +
				"Host: foo.com\r\n\r\n" +
				"hello"),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "POST",
					URL:        MustURL(t, "/some/path"),
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Host:       "foo.com",
					Header:     http.Header{},
					Body:       nil,
				}, nil, []byte("hello"),
			},
		},
		{
			name: "should respect explicit Content-Length: 0 and ignore trailing bytes",
			input: []byte("POST /some/path HTTP/1.1\r\n" +
				"Host: foo.com\r\n" +
				"Content-Length: 0\r\n\r\n" +
				"hello"),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "POST",
					URL:        MustURL(t, "/some/path"),
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Host:       "foo.com",
					Header:     http.Header{"Content-Length": []string{"0"}},
					Body:       http.NoBody,
				}, nil, nil,
			},
		},
		{
			name: "should read body larger than bufio default buffer",
			input: []byte("POST /some/path HTTP/1.1\r\n" +
				"Host: foo.com\r\n\r\n" +
				strings.Repeat("a", 8192)),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "POST",
					URL:        MustURL(t, "/some/path"),
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Host:       "foo.com",
					Header:     http.Header{},
					Body:       nil,
				}, nil, []byte(strings.Repeat("a", 8192)),
			},
		},
		{
			name: "should read binary body with CRLF inside",
			input: append([]byte("POST /some/path HTTP/1.1\r\n"+
				"Host: foo.com\r\n\r\n"),
				[]byte{0x00, 0x01, '\r', '\n', 0xff, 'X', 0x00}...),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "POST",
					URL:        MustURL(t, "/some/path"),
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Host:       "foo.com",
					Header:     http.Header{},
					Body:       nil,
				}, nil, []byte{0x00, 0x01, '\r', '\n', 0xff, 'X', 0x00},
			},
		},
		{
			name: "should accept bare LF as header separator (contract with stdlib)",
			input: []byte("GET /some/path HTTP/1.1\n" +
				"Host: foo.com\n" +
				"Content-Length: 0\n\n"),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "GET",
					URL:        MustURL(t, "/some/path"),
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Host:       "foo.com",
					Header:     http.Header{"Content-Length": []string{"0"}},
					Body:       http.NoBody,
				}, nil, nil,
			},
		},
		{
			name: "should read body after bare LF separators",
			input: []byte("POST /some/path HTTP/1.1\n" +
				"Host: foo.com\n\n" +
				"hello"),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "POST",
					URL:        MustURL(t, "/some/path"),
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Host:       "foo.com",
					Header:     http.Header{},
					Body:       nil,
				}, nil, []byte("hello"),
			},
		},
		{
			name: "should accept mixed CRLF and bare LF separators",
			input: []byte("POST /some/path HTTP/1.1\r\n" +
				"Host: foo.com\n" +
				"Foo: bar\r\n\n" +
				"hello"),
			want: DecoderRequestWant{
				&http.Request{
					Method:     "POST",
					URL:        MustURL(t, "/some/path"),
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Host:       "foo.com",
					Header:     http.Header{"Foo": []string{"bar"}},
					Body:       nil,
				}, nil, []byte("hello"),
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
				if ans.req.GetBody != nil {
					retryBody, err := ans.req.GetBody()
					assert.NoError(err)
					assert.NoError(iotest.TestReader(retryBody, tt.want.body))
				}
				ans.req.Body = nil
				ans.req.GetBody = nil
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
