// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package uri

import (
	"bufio"
	"context"
	"log"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/ammo"
)

type Config struct {
	File string `validate:"required"`
	// Limit limits total num of ammo. Unlimited if zero.
	Limit int `validate:"min=0"`
	// Passes limits ammo file passes. Unlimited if zero.
	Passes int `validate:"min=0"`
}

// TODO: pass logger and metricsRegistry
func NewProvider(fs afero.Fs, conf Config) *Provider {
	return &Provider{
		Config: conf,
		fs:     fs,
		source: make(chan ammo.Ammo, 0), // TODO: make buffer size configurable?
	}
}

type Provider struct {
	fs afero.Fs
	Config
	source chan ammo.Ammo

	decoder *decoder // Initialized on start.
}

func (p *Provider) Source() <-chan ammo.Ammo {
	return p.source
}

func (p *Provider) Release(a ammo.Ammo) {
	p.decoder.ammoPool.Put(a)
}

func (p *Provider) Start(ctx context.Context) error {
	defer close(p.source)
	p.decoder = newDecoder(p.source, ctx)

	ammoFile, err := p.fs.Open(p.File)
	if err != nil {
		return stackerr.Newf("failed to open ammo source: %v", err)
	}
	defer ammoFile.Close()

	var passNum int
	for {
		passNum++
		scanner := bufio.NewScanner(ammoFile)
		for line := 1; scanner.Scan() && (p.Limit == 0 || p.decoder.ammoNum < p.Limit); line++ {
			data := scanner.Bytes()
			err := p.decoder.Decode(data)
			if err != nil {
				if err == ctx.Err() {
					return err
				}
				return stackerr.Newf("failed to decode ammo at line: %v; data: %q; error: %s", line, data, err)
			}
		}
		if p.decoder.ammoNum == 0 {
			return stackerr.Newf("no ammo in file")
		}
		if p.Passes != 0 && passNum >= p.Passes {
			break
		}
		ammoFile.Seek(0, 0)
		p.decoder.ResetHeader()
		// TODO: set metrics?
	}
	log.Println("Ran out of ammo")
	return nil
}
