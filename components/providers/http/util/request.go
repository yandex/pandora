package util

import (
	"fmt"
	"net/http"
	"net/textproto"
	"strings"
)

var ErrHeaderFormat = fmt.Errorf("header line wrong format: expect [key: value]")
var ErrEmptyKey = fmt.Errorf("missing header key")

func DecodeHeader(h string) (key, value string, err error) {
	var ok bool
	if len(h) < 3 || h[0] != '[' || h[len(h)-1] != ']' {
		err = ErrHeaderFormat
		return
	}
	h = h[1 : len(h)-1]
	if key, value, ok = strings.Cut(h, ":"); !ok {
		err = ErrHeaderFormat
		return
	}
	key = strings.TrimSpace(key)
	if key == "" {
		err = ErrEmptyKey
		return
	}
	value = strings.TrimSpace(value)
	return
}

func DecodeHTTPConfigHeaders(headers []string) (configHTTPHeaders http.Header, err error) {
	var key, value string
	configHTTPHeaders = make(http.Header)
	for _, h := range headers {
		key, value, err = DecodeHeader(h)
		if err != nil {
			break
		}
		configHTTPHeaders.Add(key, value)
	}
	return
}

func EnrichRequestWithHeaders(req *http.Request, headers http.Header) {
	for key, values := range headers {
		key = textproto.CanonicalMIMEHeaderKey(key)
		if _, ok := req.Header[key]; !ok {
			// special behavior for `Host` header
			if key == "Host" {
				if req.Host == "" {
					req.Host = values[0]
				}
			} else {
				req.Header[key] = values
			}
		}
	}
}
