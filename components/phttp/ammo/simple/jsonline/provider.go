// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package jsonline

import (
	"bufio"
	"context"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/phttp/ammo/simple"
)

func NewProvider(fs afero.Fs, conf Config) *Provider {
	var p Provider
	p = Provider{
		Provider: simple.NewProvider(fs, conf.File, p.start),
		Config:   conf,
	}
	return &p
}

type Provider struct {
	simple.Provider
	Config
}

type Config struct {
	File string `validate:"required"`
	// Limit limits total num of ammo. Unlimited if zero.
	Limit int `validate:"min=0"`
	// Passes limits ammo file passes. Unlimited if zero.
	Passes int `validate:"min=0"`
	ContinueOnError bool
}

func (p *Provider) start(ctx context.Context, ammoFile afero.File) error {
	var ammoNum, passNum int
	for {
		passNum++
		scanner := bufio.NewScanner(ammoFile)
		for line := 1; scanner.Scan() && (p.Limit == 0 || ammoNum < p.Limit); line++ {
			data := scanner.Bytes()
			a, err := decodeAmmo(data, p.Pool.Get().(*simple.Ammo))
			if err != nil {
				if p.Config.ContinueOnError == true {
					a.Invalidate()
				} else {
					return errors.Wrapf(err, "failed to decode ammo at line: %v; data: %q", line, data)
				}
			}
			ammoNum++
			select {
			case p.Sink <- a:
			case <-ctx.Done():
				return nil
			}
		}
		if p.Passes != 0 && passNum >= p.Passes {
			break
		}
		ammoFile.Seek(0, 0)
	}
	return nil
}

func decodeAmmo(jsonDoc []byte, am *simple.Ammo) (*simple.Ammo, error) {
	var data data
	err := data.UnmarshalJSON(jsonDoc)
	if err != nil {
		return am, errors.WithStack(err)
	}
	req, err := data.ToRequest()
	if err != nil {
		return am, err
	}
	am.Reset(req, data.Tag)
	return am, nil
}

func (d *data) ToRequest() (req *http.Request, err error) {
	uri := "http://" + d.Host + d.Uri
	req, err = http.NewRequest(d.Method, uri, strings.NewReader(d.Body))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for k, v := range d.Headers {
		req.Header.Set(k, v)
	}
	return
}
