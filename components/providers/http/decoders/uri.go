package decoders

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders/ammo"
	"github.com/yandex/pandora/components/providers/http/util"
	"github.com/yandex/pandora/core"
)

func newURIDecoder(file io.ReadSeeker, cfg config.Config, decodedConfigHeaders http.Header) *uriDecoder {
	return &uriDecoder{
		protoDecoder: protoDecoder{
			file:                 file,
			config:               cfg,
			decodedConfigHeaders: decodedConfigHeaders,
		},
		scanner: bufio.NewScanner(file),
		Header:  http.Header{},
		pool:    &sync.Pool{New: func() any { return &ammo.Ammo{} }},
	}
}

type uriDecoder struct {
	protoDecoder
	scanner *bufio.Scanner
	Header  http.Header
	line    uint
	pool    *sync.Pool
}

func (d *uriDecoder) readLine(data string, commonHeader http.Header) (DecodedAmmo, error) {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		return nil, nil // skip empty line
	}
	if data[0] == '[' {
		key, val, err := util.DecodeHeader(data)
		if err != nil {
			err = fmt.Errorf("decoding header error: %w", err)
			return nil, err
		}
		commonHeader.Set(key, val)
		return nil, nil
	}

	var rawURL string
	rawURL, tag, _ := strings.Cut(data, " ")
	_, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	header := commonHeader.Clone()
	for k, vv := range d.decodedConfigHeaders {
		for _, v := range vv {
			header.Set(k, v)
		}
	}
	a := d.pool.Get().(*ammo.Ammo)
	if err := a.Setup("GET", rawURL, nil, header, tag); err != nil {
		return nil, err
	}
	return a, nil
}

func (d *uriDecoder) Release(a core.Ammo) {
	if am, ok := a.(*ammo.Ammo); ok {
		am.Reset()
		d.pool.Put(*am)
	}
}

func (d *uriDecoder) LoadAmmo(ctx context.Context) ([]DecodedAmmo, error) {
	return d.protoDecoder.LoadAmmo(ctx, d.Scan)
}

func (d *uriDecoder) Scan(ctx context.Context) (DecodedAmmo, error) {
	if d.config.Limit != 0 && d.ammoNum >= d.config.Limit {
		return nil, ErrAmmoLimit
	}
	for ; ; d.line++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if !d.scanner.Scan() {
			if d.scanner.Err() == nil { // assume as io.EOF; FIXME: check possible nil error with other reason
				d.line = 0
				d.passNum++
				if d.config.Passes != 0 && d.passNum >= d.config.Passes {
					return nil, ErrPassLimit
				}
				if d.ammoNum == 0 {
					return nil, ErrNoAmmo
				}
				d.Header = http.Header{}
				_, err := d.file.Seek(0, io.SeekStart)
				if err != nil {
					return nil, err
				}
				d.scanner = bufio.NewScanner(d.file)
				continue
			}
			return nil, d.scanner.Err()
		}
		data := d.scanner.Text()
		a, err := d.readLine(data, d.Header)
		if err != nil {
			return nil, fmt.Errorf("decode at line %d `%s` error: %w", d.line+1, data, err)
		}
		if a != nil {
			d.ammoNum++
			return a, nil
		}
	}
}
