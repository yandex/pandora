// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package uri

import (
	"bufio"
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	simple "github.com/yandex/pandora/components/providers/http"
	"go.uber.org/zap"
)

type Config struct {
	File string
	// Limit limits total num of ammo. Unlimited if zero.
	Limit int `validate:"min=0"`
	// Additional HTTP headers
	Headers []string
	// Passes limits ammo file passes. Unlimited if zero.
	Passes      int `validate:"min=0"`
	Uris        []string
	ChosenCases []string
}

// TODO: pass logger and metricsRegistry
func NewProvider(fs afero.Fs, conf Config) *Provider {
	if len(conf.Uris) > 0 {
		if conf.File != "" {
			panic(`One should specify either 'file' or 'uris', but not both of them.`)
		}
		file, err := afero.TempFile(fs, "", "generated_ammo_")
		if err != nil {
			panic(fmt.Sprintf(`failed to create tmp ammo file: %v`, err))
		}
		for _, uri := range conf.Uris {
			_, err := file.WriteString(fmt.Sprintf("%s\n", uri))
			if err != nil {
				panic(fmt.Sprintf(`failed to write ammo in tmp file: %v`, err))
			}
		}
		conf.File = file.Name()
	}
	if conf.File == "" {
		panic(`One should specify either 'file' or 'uris'.`)
	}
	var p Provider
	p = Provider{
		Provider: simple.NewProvider(fs, conf.File, p.start),
		Config:   conf,
	}
	p.Close = func() {
		if len(conf.Uris) > 0 {
			err := fs.Remove(conf.File)
			if err != nil {
				zap.L().Error("failed to delete temp file", zap.String("file name", conf.File))
			}
		}
	}
	return &p
}

type Provider struct {
	simple.Provider
	Config
	log *zap.Logger

	decoder *decoder // Initialized on start.
}

func (p *Provider) start(ctx context.Context, ammoFile afero.File) error {
	p.decoder = newDecoder(ctx, p.Sink, &p.Pool, p.Config.ChosenCases)
	// parse and prepare Headers from config
	decodedConfigHeaders, err := simple.DecodeHTTPConfigHeaders(p.Config.Headers)
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
				if errors.Is(err, context.Canceled) {
					return context.Canceled
				}
				return errors.Wrapf(err, "failed to decode ammo at line: %v; data: %q", line, data)
			}
		}
		if p.decoder.ammoNum == 0 {
			return errors.New("no ammo in file")
		}
		if p.Passes != 0 && passNum >= p.Passes {
			break
		}
		_, err := ammoFile.Seek(0, 0)
		if err != nil {
			p.log.Info("Failed to seek ammo file", zap.Error(err))
		}
		p.decoder.ResetHeader()
	}
	return nil
}
