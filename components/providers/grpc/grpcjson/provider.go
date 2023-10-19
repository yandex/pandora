// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package grpcjson

import (
	"bufio"
	"context"

	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	ammo "github.com/yandex/pandora/components/providers/grpc"
	"github.com/yandex/pandora/lib/confutil"
	"go.uber.org/zap"
)

func NewProvider(fs afero.Fs, conf Config) *Provider {
	var p Provider
	if conf.Source.Path != "" {
		conf.File = conf.Source.Path
	}
	p = Provider{
		Provider: ammo.NewProvider(fs, conf.File, p.start),
		Config:   conf,
	}
	return &p
}

type Provider struct {
	ammo.Provider
	Config
	log *zap.Logger
}

type Source struct {
	Type string
	Path string
}

type Config struct {
	File string //`validate:"required"`
	// Limit limits total num of ammo. Unlimited if zero.
	Limit int `validate:"min=0"`
	// Passes limits ammo file passes. Unlimited if zero.
	Passes          int `validate:"min=0"`
	ContinueOnError bool
	//Maximum number of byte in an ammo. Default is bufio.MaxScanTokenSize
	MaxAmmoSize int
	Source      Source `config:"source"`
	ChosenCases []string
}

func (p *Provider) start(ctx context.Context, ammoFile afero.File) error {
	var ammoNum, passNum int
	for {
		passNum++
		scanner := bufio.NewScanner(ammoFile)
		if p.Config.MaxAmmoSize != 0 {
			var buffer []byte
			scanner.Buffer(buffer, p.Config.MaxAmmoSize)
		}
		for line := 1; scanner.Scan() && (p.Limit == 0 || ammoNum < p.Limit); line++ {
			data := scanner.Bytes()
			a, err := decodeAmmo(data, p.Pool.Get().(*ammo.Ammo))
			if err != nil {
				if p.Config.ContinueOnError {
					a.Invalidate()
				} else {
					return errors.Wrapf(err, "failed to decode ammo at line: %v; data: %q", line, data)
				}
			}
			if !confutil.IsChosenCase(a.Tag, p.Config.ChosenCases) {
				continue
			}
			ammoNum++
			select {
			case p.Sink <- a:
			case <-ctx.Done():
				return nil
			}
		}
		err := scanner.Err()
		if err != nil {
			return errors.Wrap(err, "gPRC Provider scan() err")
		}
		if p.Passes != 0 && passNum >= p.Passes {
			break
		}
		_, err = ammoFile.Seek(0, 0)
		if err != nil {
			return errors.Wrap(err, "Failed to seek ammo file")
		}
	}
	return nil
}

func decodeAmmo(jsonDoc []byte, am *ammo.Ammo) (*ammo.Ammo, error) {
	var ammo ammo.Ammo
	err := jsoniter.Unmarshal(jsonDoc, &ammo)
	if err != nil {
		return am, errors.WithStack(err)
	}

	am.Reset(ammo.Tag, ammo.Call, ammo.Metadata, ammo.Payload)
	return am, nil
}
