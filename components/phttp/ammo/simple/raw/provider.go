package raw

import (
	"bufio"
	"context"
	"io"
	"log"
	"os"

	"github.com/facebookgo/stackerr"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/components/phttp/ammo/simple"
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

func (p *Provider) start(ctx context.Context, ammoFile afero.File) error {
	var passNum int
	var ammoNum int
	for {
		passNum++
		reader := bufio.NewReader(ammoFile)
		for p.Limit == 0 || ammoNum < p.Limit {
			data, isPrefix, err := reader.ReadLine()
			if isPrefix {
				offset, _ := ammoFile.Seek(0, os.SEEK_CUR)
				return stackerr.Newf("Too long header in ammo at position %v", offset)
			}
			if err == io.EOF {
				break // start over from the beginning
			}
			if err != nil {
				if err == ctx.Err() {
					return err
				}
				offset, _ := ammoFile.Seek(0, os.SEEK_CUR)
				return stackerr.Newf("error reading ammo at position: %v; error: %s", offset, err)
			}
			if len(data) == 0 {
				continue // skip empty lines
			}
			reqSize, tag, err := decodeHeader(data)
			if reqSize == 0 {
				break // start over from the beginning of file if ammo size is 0
			}
			buff := make([]byte, reqSize)
			reader.Read(buff)
			req, err := decodeRequest(buff)
			if err != nil {
				if err == ctx.Err() {
					return err
				}
				offset, _ := ammoFile.Seek(0, os.SEEK_CUR)
				return stackerr.Newf("failed to decode ammo at position: %v; data: %q; error: %s", offset, buff, err)
			}
			sh := p.Pool.Get().(*simple.Ammo)
			sh.Reset(req, tag)

			select {
			case p.Sink <- sh:
				ammoNum++
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if ammoNum == 0 {
			return stackerr.Newf("no ammo in file")
		}
		if p.Passes != 0 && passNum >= p.Passes {
			break
		}
		ammoFile.Seek(0, 0)
	}
	log.Println("Ran out of ammo")
	return nil
}
