// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package uri

import (
	"bufio"
	"context"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/components/phttp/ammo/simple"
)

type Config struct {
	File string `validate:"required"`
	// Limit limits total num of ammo. Unlimited if zero.
	Limit int `validate:"min=0"`
	// Redefine HTTP headers
	Headers []string
	// Passes limits ammo file passes. Unlimited if zero.
	Passes int `validate:"min=0"`
}

// TODO: pass logger and metricsRegistry
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

	decoder *decoder // Initialized on start.
}

func (p *Provider) start(ctx context.Context, ammoFile afero.File) error {
	p.decoder = newDecoder(ctx, p.Sink, &p.Pool)
	// parse and prepare Headers from config
	decodedConfigHeaders, err := decodeHTTPConfigHeaders(p.Config.Headers)
	if err != nil {
		return err
	}
	p.decoder.configHeaders = decodedConfigHeaders
	var passNum int
	for {
		passNum++
		scanner := bufio.NewScanner(ammoFile)
		for line := 1; scanner.Scan() && (p.Limit == 0 || p.decoder.ammoNum < p.Limit); line++ {
			data := scanner.Bytes()
			if len(data) == 0 {
				continue // skip empty lines
			}
			err := p.decoder.Decode(data)
			if err != nil {
				return errors.Wrapf(err, "failed to decode ammo at line: %v; data: %q", line, data)
			}
		}
		if p.decoder.ammoNum == 0 {
			return errors.New("no ammo in file")
		}
		if p.Passes != 0 && passNum >= p.Passes {
			break
		}
		ammoFile.Seek(0, 0)
		p.decoder.ResetHeader()
	}
	return nil
}
