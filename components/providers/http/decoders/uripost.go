package decoders

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders/uripost"
	"github.com/yandex/pandora/components/providers/http/util"
	"golang.org/x/xerrors"
)

func newURIPostDecoder(file io.ReadSeeker, cfg config.Config, decodedConfigHeaders http.Header) *uripostDecoder {
	return &uripostDecoder{
		protoDecoder: protoDecoder{
			file:                 file,
			config:               cfg,
			decodedConfigHeaders: decodedConfigHeaders,
		},
		reader: bufio.NewReader(file),
		header: http.Header{},
	}
}

type uripostDecoder struct {
	protoDecoder
	reader *bufio.Reader
	header http.Header
	line   uint
}

func (d *uripostDecoder) Scan(ctx context.Context) (*http.Request, string, error) {
	if d.config.Limit != 0 && d.ammoNum >= d.config.Limit {
		return nil, "", ErrAmmoLimit
	}
	for i := 0; i < 2; i++ {
		for {
			if ctx.Err() != nil {
				return nil, "", ctx.Err()
			}

			req, tag, err := d.readBlock(d.reader, d.header)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, "", err
			}
			if req != nil {
				d.ammoNum++
				return req, tag, nil
			}
			// here only if read header
		}

		// seek file
		d.passNum++
		if d.config.Passes != 0 && d.passNum >= d.config.Passes {
			return nil, "", ErrPassLimit
		}
		if d.ammoNum == 0 {
			return nil, "", ErrNoAmmo
		}
		d.header = make(http.Header)
		_, err := d.file.Seek(0, io.SeekStart)
		if err != nil {
			return nil, "", err
		}
		d.reader.Reset(d.file)
	}

	return nil, "", errors.New("unexpected behavior")
}

// readBlock read one header at time and set to commonHeader or read full request
func (d *uripostDecoder) readBlock(reader *bufio.Reader, commonHeader http.Header) (*http.Request, string, error) {
	data, err := reader.ReadString('\n')
	if err != nil {
		return nil, "", err
	}
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		return nil, "", nil // skip empty lines
	}
	if data[0] == '[' {
		key, val, err := util.DecodeHeader(data)
		if err != nil {
			return nil, "", err
		}
		commonHeader.Set(key, val)
		return nil, "", nil
	}

	bodySize, uri, tag, err := uripost.DecodeURI(data)
	if err != nil {
		return nil, "", err
	}

	var buffReader io.Reader
	buff := make([]byte, bodySize)
	if bodySize != 0 {
		if n, err := io.ReadFull(reader, buff); err != nil {
			err = xerrors.Errorf("failed to read ammo with err: %w, at position: %v; tried to read: %v; have read: %v", err, filePosition(d.file), bodySize, n)
			return nil, "", err
		}
		buffReader = bytes.NewReader(buff)
	}
	req, err := http.NewRequest("POST", uri, buffReader)
	if err != nil {
		err = xerrors.Errorf("failed to decode ammo with err: %w, at position: %v; data: %q", err, filePosition(d.file), buff)
		return nil, "", err
	}

	for k, v := range commonHeader {
		// http.Request.Write sends Host header based on req.URL.Host
		if k == "Host" {
			req.Host = v[0]
		} else {
			req.Header[k] = v
		}
	}

	// add new Headers to request from config
	util.EnrichRequestWithHeaders(req, d.decodedConfigHeaders)

	return req, tag, nil
}
