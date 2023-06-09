package decoders

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders/jsonline"
	"github.com/yandex/pandora/components/providers/http/util"
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
	}
}

type jsonlineDecoder struct {
	protoDecoder
	scanner *bufio.Scanner
	line    uint
}

func (d *jsonlineDecoder) Scan(ctx context.Context) (*http.Request, string, error) {
	if d.config.Limit != 0 && d.ammoNum >= d.config.Limit {
		return nil, "", ErrAmmoLimit
	}
	for ; ; d.line++ {
		if ctx.Err() != nil {
			return nil, "", ctx.Err()
		}

		if !d.scanner.Scan() {
			if d.scanner.Err() == nil { // assume as io.EOF; FIXME: check possible nil error with other reason
				d.line = 0
				d.passNum++
				if d.config.Passes != 0 && d.passNum >= d.config.Passes {
					return nil, "", ErrPassLimit
				}
				if d.ammoNum == 0 {
					return nil, "", ErrNoAmmo
				}
				_, err := d.file.Seek(0, io.SeekStart)
				if err != nil {
					return nil, "", err
				}
				d.scanner = bufio.NewScanner(d.file)
				if d.config.MaxAmmoSize != 0 {
					var buffer []byte
					d.scanner.Buffer(buffer, d.config.MaxAmmoSize)
				}
				continue
			}
			return nil, "", d.scanner.Err()
		}
		data := d.scanner.Bytes()
		if len(strings.TrimSpace(string(data))) == 0 {
			continue
		}
		d.ammoNum++

		req, tag, err := jsonline.DecodeAmmo(data)
		if err != nil {
			if !d.config.ContinueOnError {
				return nil, "", xerrors.Errorf("failed to decode ammo at line: %v; data: %q, with err: %w", d.line+1, data, err)
			}
			// TODO: add log message about error
			continue // skipping ammo
		}
		util.EnrichRequestWithHeaders(req, d.decodedConfigHeaders)
		return req, tag, err
	}
}
