package decoders

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders/ammo"
	"github.com/yandex/pandora/core"
	"golang.org/x/xerrors"
)

func newJsonlineDecoder(file io.ReadSeeker, cfg config.Config, decodedConfigHeaders http.Header) (*jsonlineDecoder, error) {
	scanner := bufio.NewScanner(file)
	if cfg.MaxAmmoSize != 0 {
		var buffer []byte
		scanner.Buffer(buffer, cfg.MaxAmmoSize)
	}

	isArray, err := isArray(file)
	if err != nil {
		return nil, err
	}

	decoder := &jsonlineDecoder{
		protoDecoder: protoDecoder{
			file:                 file,
			config:               cfg,
			decodedConfigHeaders: decodedConfigHeaders,
		},
		scanner: scanner,
		pool:    &sync.Pool{New: func() any { return &ammo.Ammo{} }},
		decoder: json.NewDecoder(file),
	}
	if isArray {
		ammos, err := decoder.readArray()
		if err != nil {
			return decoder, fmt.Errorf("cant read json array: %w", err)
		}
		decoder.ammos = ammos
	}
	return decoder, nil
}

func isArray(r io.ReadSeeker) (bool, error) {
	d := json.NewDecoder(r)
	t, err := d.Token()
	if err != nil {
		return false, err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return false, errors.New("invalid json token")
	}
	_, err = r.Seek(0, io.SeekStart)
	return delim.String() == "[", err
}

type jsonlineDecoder struct {
	protoDecoder
	scanner *bufio.Scanner
	line    uint
	pool    *sync.Pool
	decoder *json.Decoder
	ammos   []DecodedAmmo
}

func (d *jsonlineDecoder) Release(a core.Ammo) {
	if am, ok := a.(*ammo.Ammo); ok {
		am.Reset()
		d.pool.Put(am)
	}
}

func (d *jsonlineDecoder) LoadAmmo(ctx context.Context) ([]DecodedAmmo, error) {
	return d.protoDecoder.LoadAmmo(ctx, d.Scan)
}

type entity struct {
	// Host defines Host header to send.
	// Request endpoint is defied by gun config.
	Host   string `json:"host"`
	Method string `json:"method"`
	URI    string `json:"uri"`
	// Headers defines headers to send.
	// NOTE: Host header will be silently ignored.
	Headers map[string]string `json:"headers"`
	Tag     string            `json:"tag"`
	// Body should be string, doublequotes should be escaped for json body
	Body string `json:"body"`
}

func (d *jsonlineDecoder) Scan(ctx context.Context) (DecodedAmmo, error) {
	if d.config.Limit != 0 && d.ammoNum >= d.config.Limit {
		return nil, ErrAmmoLimit
	}
	if d.ammos != nil {
		return d.scanAmmos()
	}
	for {
		if d.config.Passes != 0 && d.passNum >= d.config.Passes {
			return nil, ErrPassLimit
		}
		var da entity
		err := d.decoder.Decode(&da)
		if err != nil {
			if err != io.EOF {
				return nil, xerrors.Errorf("failed to decode ammo at line: %v; with err: %w", d.line+1, err)
			}
			// go to next pass
		} else {
			d.line++
			d.ammoNum++

			header := d.decodedConfigHeaders.Clone()
			for k, v := range da.Headers {
				header.Set(k, v)
			}
			url := "http://" + da.Host + da.URI // schema will be rewrite in gun
			var body []byte
			if da.Body != "" {
				body = []byte(da.Body)
			}
			a := d.pool.Get().(*ammo.Ammo)
			err = a.Setup(da.Method, url, body, header, da.Tag)
			return a, err
		}

		err = d.scanner.Err()
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
		d.decoder = json.NewDecoder(d.file)
	}
}

func (d *jsonlineDecoder) readArray() ([]DecodedAmmo, error) {
	var data []entity
	err := d.decoder.Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("cant readArray, err: %w", err)
	}
	result := make([]DecodedAmmo, len(data))
	for i, datum := range data {
		header := d.decodedConfigHeaders.Clone()
		for k, v := range datum.Headers {
			header.Set(k, v)
		}
		url := "http://" + datum.Host + datum.URI // schema will be rewrite in gun
		var body []byte
		if datum.Body != "" {
			body = []byte(datum.Body)
		}
		a := d.pool.Get().(*ammo.Ammo)
		err = a.Setup(datum.Method, url, body, header, datum.Tag)
		if err != nil {
			return nil, fmt.Errorf("cant readArray, err: %w", err)
		}
		result[i] = a
	}

	return result, nil
}

func (d *jsonlineDecoder) scanAmmos() (DecodedAmmo, error) {
	length := len(d.ammos)
	if length == 0 {
		return nil, ErrNoAmmo
	}
	if d.config.Passes != 0 && d.passNum >= d.config.Passes {
		return nil, ErrPassLimit
	}
	i := int(d.ammoNum) % length
	a := d.ammos[i]
	if d.ammoNum > 0 && i == length-1 {
		d.passNum++
	}
	d.ammoNum++
	return a, nil
}
