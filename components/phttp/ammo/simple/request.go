// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package simple

import (
	"bytes"
	"net/http"

	"github.com/pkg/errors"
)

type Header struct {
	key   string
	value string
}

func DecodeHTTPConfigHeaders(headers []string) (configHTTPHeaders []Header, err error) {
	for _, header := range headers {
		line := []byte(header)
		if len(line) < 3 || line[0] != '[' || line[len(line)-1] != ']' {
			return nil, errors.New("header line should be like '[key: value]")
		}
		line = line[1 : len(line)-1]
		colonIdx := bytes.IndexByte(line, ':')
		if colonIdx < 0 {
			return nil, errors.New("missing colon")
		}
		configHTTPHeaders = append(
			configHTTPHeaders,
			Header{
				string(bytes.TrimSpace(line[:colonIdx])),
				string(bytes.TrimSpace(line[colonIdx+1:])),
			})
	}
	return
}

func UpdateRequestWithHeaders(req *http.Request, headers []Header) {
	origHeaders := req.Header.Clone()
	for _, header := range headers {
		if origHeaders.Get(header.key) != "" {
			continue
		}
		// special behavior for `Host` header
		if header.key == "Host" {
			if req.Host == "" {
				req.Host = header.value
			}
		} else {
			req.Header.Add(header.key, header.value)
		}
	}
}
