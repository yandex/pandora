// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package uri

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"sync"

	"github.com/pkg/errors"

	"github.com/yandex/pandora/components/phttp/ammo/simple"
)

type decoder struct {
	ctx  context.Context
	sink chan<- *simple.Ammo
	pool *sync.Pool

	ammoNum       int
	header        http.Header
	configHeaders []ConfigHeader
}

type ConfigHeader struct {
	key   string
	value string
}

func newDecoder(ctx context.Context, sink chan<- *simple.Ammo, pool *sync.Pool) *decoder {
	return &decoder{
		sink:   sink,
		header: http.Header{},
		pool:   pool,
		ctx:    ctx,
	}
}

func (d *decoder) Decode(line []byte) error {
	// FIXME: rewrite decoder
	// OPTIMIZE: reuse *http.Request, http.Header. Benchmark both variants.
	if len(line) == 0 {
		return errors.New("empty line")
	}
	line = bytes.TrimSpace(line)
	switch line[0] {
	case '/':
		return d.decodeURI(line)
	case '[':
		return d.decodeHeader(line)
	}
	return errors.New("every line should begin with '[' or '/'")
}

func (d *decoder) decodeURI(line []byte) error {
	parts := strings.SplitN(string(line), " ", 2)
	url := parts[0]
	var tag string
	if len(parts) > 1 {
		tag = parts[1]
	}
	req, err := http.NewRequest("GET", string(url), nil)
	if err != nil {
		return errors.Wrap(err, "uri decode")
	}
	for k, v := range d.header {
		// http.Request.Write sends Host header based on req.URL.Host
		if k == "Host" {
			req.Host = v[0]
			req.URL.Host = v[0]
		} else {
			req.Header[k] = v
		}
	}
	// redefine request Headers from config
	for _, configHeader := range d.configHeaders {
		if configHeader.key == "Host" {
			req.Host = configHeader.value
			req.URL.Host = configHeader.value
		} else {
			req.Header.Set(configHeader.key, configHeader.value)
		}
	}

	sh := d.pool.Get().(*simple.Ammo)
	sh.Reset(req, tag)
	select {
	case d.sink <- sh:
		d.ammoNum++
		return nil
	case <-d.ctx.Done():
		return d.ctx.Err()
	}
}

func (d *decoder) decodeHeader(line []byte) error {
	if len(line) < 3 || line[0] != '[' || line[len(line)-1] != ']' {
		return errors.New("header line should be like '[key: value]")
	}
	line = line[1 : len(line)-1]
	colonIdx := bytes.IndexByte(line, ':')
	if colonIdx < 0 {
		return errors.New("missing colon")
	}
	key := string(bytes.TrimSpace(line[:colonIdx]))
	val := string(bytes.TrimSpace(line[colonIdx+1:]))
	if key == "" {
		return errors.New("missing header key")
	}
	d.header.Set(key, val)
	return nil
}

func (d *decoder) ResetHeader() {
	for k := range d.header {
		delete(d.header, k)
	}
}

func decodeHTTPConfigHeaders(headers []string) (configHTTPHeaders []ConfigHeader, err error) {
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
		preparedHeader := ConfigHeader{
			string(bytes.TrimSpace(line[:colonIdx])),
			string(bytes.TrimSpace(line[colonIdx+1:])),
		}
		configHTTPHeaders = append(configHTTPHeaders, preparedHeader)
	}
	return
}
