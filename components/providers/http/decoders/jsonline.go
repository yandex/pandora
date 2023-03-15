package decoders

import (
	"bufio"
	"context"
	"io"

	"github.com/yandex/pandora/components/providers/http/decoders/jsonline"
	"github.com/yandex/pandora/components/providers/http/util"
	"golang.org/x/xerrors"
)

type jsonlineDecoder struct {
	protoDecoder
	scanner *bufio.Scanner
	line    uint
}

func (d *jsonlineDecoder) Scan(ctx context.Context) bool {
	if d.Limit != 0 && d.ammoNum >= d.Limit {
		d.err = ErrAmmoLimit
		return false
	}
	if d.Config.Limit != 0 && d.ammoNum >= d.Config.Limit {
		d.err = ErrAmmoLimit
		return false
	}
	for ; ; d.line++ {
		select {
		case <-ctx.Done():
			d.err = ctx.Err()
			return false
		default:
		}
		if !d.scanner.Scan() {
			if d.scanner.Err() == nil { // assume as io.EOF; FIXME: check possible nil error with other reason
				d.line = 0
				d.passNum++
				if d.Passes != 0 && d.passNum >= d.Passes {
					d.err = ErrPassLimit
					return false
				}
				if d.ammoNum == 0 {
					d.err = ErrNoAmmo
					return false
				}
				_, err := d.file.Seek(0, io.SeekStart)
				if err != nil {
					d.err = err
					return false
				}
				d.scanner = bufio.NewScanner(d.file)
				if d.Config.MaxAmmoSize != 0 {
					var buffer []byte
					d.scanner.Buffer(buffer, d.Config.MaxAmmoSize)
				}
				continue
			}
			d.err = d.scanner.Err()
			return false
		}
		data := d.scanner.Bytes()
		d.ammoNum++
		var err error
		d.req, d.tag, err = jsonline.DecodeAmmo(data)
		if err != nil {
			if !d.Config.ContinueOnError {
				d.err = xerrors.Errorf("failed to decode ammo at line: %v; data: %q, with err: %w", d.line+1, data, err)
				return false
			}
			// TODO: add log message about error
			continue // skipping ammo
		}
		util.EnrichRequestWithHeaders(d.req, d.decodedConfigHeaders)
		return true
	}
}
