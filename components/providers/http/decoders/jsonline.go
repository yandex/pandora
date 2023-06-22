package decoders

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders/ammo"
	"github.com/yandex/pandora/components/providers/http/decoders/jsonline"
	"github.com/yandex/pandora/core"
	"golang.org/x/xerrors"
)

func newJsonlineDecoder(file io.ReadSeeker, cfg config.Config, decodedConfigHeaders http.Header) *jsonlineDecoder {
	scanner := bufio.NewScanner(file)
	if cfg.MaxAmmoSize != 0 {
		var buffer []byte
		scanner.Buffer(buffer, cfg.MaxAmmoSize)
	}
	return &jsonlineDecoder{
		protoDecoder: protoDecoder{
			file:                 file,
			config:               cfg,
			decodedConfigHeaders: decodedConfigHeaders,
		},
		scanner: scanner,
		pool:    &sync.Pool{New: func() any { return &ammo.Ammo{} }},
	}
}

type jsonlineDecoder struct {
	protoDecoder
	scanner *bufio.Scanner
	line    uint
	pool    *sync.Pool
}

func (d *jsonlineDecoder) Release(a core.Ammo) {
	if am, ok := a.(*ammo.Ammo); ok {
		am.Reset()
		d.pool.Put(am)
	}
}

func (d *jsonlineDecoder) Scan(ctx context.Context) (DecodedAmmo, error) {
	if d.config.Limit != 0 && d.ammoNum >= d.config.Limit {
		return nil, ErrAmmoLimit
	}
	for {
		if d.config.Passes != 0 && d.passNum >= d.config.Passes {
			return nil, ErrPassLimit
		}

		for d.scanner.Scan() {
			d.line++
			data := d.scanner.Bytes()
			if len(strings.TrimSpace(string(data))) == 0 {
				continue
			}
			d.ammoNum++
			method, url, header, tag, body, err := jsonline.DecodeAmmo(data, d.decodedConfigHeaders)
			if err != nil {
				if !d.config.ContinueOnError {
					return nil, xerrors.Errorf("failed to decode ammo at line: %v; data: %q, with err: %w", d.line+1, data, err)
				}
				// TODO: add log message about error
				continue // skipping ammo
			}
			a := d.pool.Get().(*ammo.Ammo)
			err = a.Setup(method, url, body, header, tag)
			return a, err
		}

		err := d.scanner.Err()
		if err != nil {
			return nil, err
		}
		if d.ammoNum == 0 {
			return nil, ErrNoAmmo
		}
		d.line = 0
		d.passNum++

		_, err = d.file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, err
		}
		d.scanner = bufio.NewScanner(d.file)
		if d.config.MaxAmmoSize != 0 {
			var buffer []byte
			d.scanner.Buffer(buffer, d.config.MaxAmmoSize)
		}
	}
}
