package decoders

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/yandex/pandora/components/providers/http/util"
	"golang.org/x/xerrors"
)

type uriDecoder struct {
	protoDecoder
	scanner *bufio.Scanner
	http.Header
	line uint
}

func (d *uriDecoder) Scan(ctx context.Context) bool {
	if d.Limit != 0 && d.ammoNum >= d.Limit {
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
				d.Header = make(http.Header)
				_, err := d.file.Seek(0, io.SeekStart)
				if err != nil {
					d.err = err
					return false
				}
				d.scanner = bufio.NewScanner(d.file)
				continue
			}
			d.err = d.scanner.Err()
			return false
		}
		data := strings.TrimSpace(d.scanner.Text())
		if len(data) == 0 {
			continue // skip empty lines
		}
		if data[0] == '[' {
			key, val, err := util.DecodeHeader(data)
			if err != nil {
				d.err = xerrors.Errorf("decoding header on line %d error: %w", d.line+1, err)
				return false
			}
			d.Header.Set(key, val)
		} else {
			d.ammoNum++
			rawURL, tag, _ := strings.Cut(data, " ")
			d.tag = tag
			req, err := http.NewRequest("", rawURL, nil)
			if err != nil {
				d.err = xerrors.Errorf("failed to decode uri at line %d: %w", d.line+1, err)
				return false
			}
			if host, ok := d.Header["Host"]; ok {
				req.Host = host[0]
			}
			req.Header = d.Header.Clone()

			// add new Headers to request from config
			util.EnrichRequestWithHeaders(req, d.decodedConfigHeaders)
			d.req = req
			return true
		}
	}
}
