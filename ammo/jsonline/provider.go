// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package jsonline

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/ammo"
)

// TODO: pass logger and metricsRegistry
func NewProvider(fs afero.Fs, conf Config) *Provider {
	return &Provider{
		Config: conf,
		fs:     fs,
		DecodeProvider: ammo.NewDecodeProvider(
			0,
			&decoder{},
			func() interface{} { return &ammo.SimpleHTTP{} },
		),
	}
}

type Provider struct {
	*ammo.DecodeProvider
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

var _ ammo.Decoder = decoder{}

func (d decoder) Decode(jsonDoc []byte, am ammo.Ammo) (ammo.Ammo, error) {
	var data data
	err := data.UnmarshalJSON(jsonDoc)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}
	req, err := data.ToRequest()
	if err != nil {
		return nil, err
	}
	sh := am.(*ammo.SimpleHTTP)
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

func (ap *Provider) Start(ctx context.Context) error {
	defer close(ap.Sink)
	ammoFile, err := ap.fs.Open(ap.File)
	if err != nil {
		return fmt.Errorf("failed to open ammo source: %v", err)
	}
	defer ammoFile.Close()
	ammoNumber := 0
	passNum := 0
	for {
		passNum++
		scanner := bufio.NewScanner(ammoFile)
		for scanner.Scan() && (ap.Limit == 0 || ammoNumber < ap.Limit) {
			data := scanner.Bytes()
			if a, err := ap.Decode(data); err != nil {
				return fmt.Errorf("failed to decode ammo: %v", err)
			} else {
				ammoNumber++
				select {
				case ap.Sink <- a:
				case <-ctx.Done():
					log.Printf("Context error: %s", ctx.Err())
					return ctx.Err()
				}
			}
		}
		if ap.Passes != 0 && passNum >= ap.Passes {
			break
		}
		ammoFile.Seek(0, 0)
		// TODO: test metrics after https://github.com/yandex/pandora/issues/34 (Use rcrowley/go-metrics instead of ugly wrappers over expvar)
		if ap.Passes == 0 {
			evPassesLeft.Set(-1)
			//log.Printf("Restarted ammo from the beginning. Infinite passes.\n") // TODO: log to debug
		} else {
			evPassesLeft.Set(int64(ap.Passes - passNum))
			//log.Printf("Restarted ammo from the beginning. Passes left: %d\n", ap.passes-passNum) // TODO: log to debug
		}
	}
	log.Println("Ran out of ammo")
	return nil
}
