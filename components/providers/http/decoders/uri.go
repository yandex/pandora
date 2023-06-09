package decoders

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/util"
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
	}
}

type uriDecoder struct {
	protoDecoder
	scanner *bufio.Scanner
	Header  http.Header
	line    uint
}

func (d *uriDecoder) readLine(data string, commonHeader http.Header) (*http.Request, string, error) {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		return nil, "", nil // skip empty line
	}
	var req *http.Request
	var tag string
	var err error
	if data[0] == '[' {
		key, val, err := util.DecodeHeader(data)
		if err != nil {
			err = fmt.Errorf("decoding header error: %w", err)
			return nil, "", err
		}
		commonHeader.Set(key, val)
	} else {
		var rawURL string
		rawURL, tag, _ = strings.Cut(data, " ")
		req, err = http.NewRequest("GET", rawURL, nil)
		if err != nil {
			err = fmt.Errorf("failed to decode uri: %w", err)
			return nil, "", err
		}
		if host, ok := commonHeader["Host"]; ok {
			req.Host = host[0]
		}
		req.Header = commonHeader.Clone()

		// add new Headers to request from config
		util.EnrichRequestWithHeaders(req, d.decodedConfigHeaders)
	}
	return req, tag, nil
}

func (d *uriDecoder) Scan(ctx context.Context) (*http.Request, string, error) {
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
				d.Header = http.Header{}
				_, err := d.file.Seek(0, io.SeekStart)
				if err != nil {
					return nil, "", err
				}
				d.scanner = bufio.NewScanner(d.file)
				continue
			}
			return nil, "", d.scanner.Err()
		}
		data := d.scanner.Text()
		req, tag, err := d.readLine(data, d.Header)
		if err != nil {
			return nil, "", fmt.Errorf("decode at line %d `%s` error: %w", d.line+1, data, err)
		}
		if req != nil {
			d.ammoNum++
			return req, tag, nil
		}
	}
}
