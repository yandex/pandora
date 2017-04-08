// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package jsonline

import (
	"bufio"
	"context"
	"log"
	"net/http"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/components/phttp/ammo"
	"github.com/yandex/pandora/core"
)

// TODO: pass logger and metricsRegistry
func NewProvider(fs afero.Fs, conf Config) *Provider {
	return &Provider{
		Config: conf,
		fs:     fs,
		DecodeProvider: NewDecodeProvider(
			0,
			&decoder{},
			func() interface{} { return &ammo.Simple{} },
		),
	}
}

type Provider struct {
	*DecodeProvider
	Config
	fs afero.Fs
}

type Config struct {
	File string `validate:"required"`
	// Limit limits total num of ammo. Unlimited if zero.
	Limit int `validate:"min=0"`
	// Passes limits ammo file passes. Unlimited if zero.
	Passes int `validate:"min=0"`
}

type decoder struct{}

var _ Decoder = decoder{}

func (d decoder) Decode(jsonDoc []byte, am core.Ammo) (core.Ammo, error) {
	var data data
	err := data.UnmarshalJSON(jsonDoc)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	req, err := data.ToRequest()
	if err != nil {
		return nil, err
	}
	sh := am.(*ammo.Simple)
	sh.Reset(req, data.Tag)
	return am, nil
}

func (d *data) ToRequest() (req *http.Request, err error) {
	uri := "http://" + d.Host + d.Uri
	req, err = http.NewRequest(d.Method, uri, nil)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	for k, v := range d.Headers {
		req.Header.Set(k, v)
	}
	return
}

func (p *Provider) Start(ctx context.Context) error {
	defer close(p.Sink)
	ammoFile, err := p.fs.Open(p.File)
	if err != nil {
		return stackerr.Newf("failed to open ammo source: %v", err)
	}
	defer ammoFile.Close()
	var ammoNum, passNum int
	for {
		passNum++
		scanner := bufio.NewScanner(ammoFile)
		for line := 1; scanner.Scan() && (p.Limit == 0 || ammoNum < p.Limit); line++ {
			data := scanner.Bytes()
			a, err := p.Decode(data)
			if err != nil {
				return stackerr.Newf("failed to decode ammo at line: %v; data: %q; error: %s", line, data, err)
			}
			ammoNum++
			select {
			case p.Sink <- a:
			case <-ctx.Done():
				log.Printf("Context error: %s", ctx.Err())
				return ctx.Err()
			}
		}
		if p.Passes != 0 && passNum >= p.Passes {
			break
		}
		ammoFile.Seek(0, 0)
		// TODO: test metrics after https://github.com/yandex/pandora/issues/34 (Use rcrowley/go-metrics instead of ugly wrappers over expvar)
		if p.Passes == 0 {
			evPassesLeft.Set(-1)
			//log.Printf("Restarted ammo from the beginning. Infinite passes.\n") // TODO: log to debug
		} else {
			evPassesLeft.Set(int64(p.Passes - passNum))
			//log.Printf("Restarted ammo from the beginning. Passes left: %d\n", p.passes-passNum) // TODO: log to debug
		}
	}
	log.Println("Ran out of ammo")
	return nil
}
