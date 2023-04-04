package decoders

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/yandex/pandora/components/providers/http/decoders/uripost"
	"github.com/yandex/pandora/components/providers/http/util"
	"golang.org/x/xerrors"
)

type uripostDecoder struct {
	protoDecoder
	reader *bufio.Reader
	header http.Header
	line   uint
}

func (d *uripostDecoder) Scan(ctx context.Context) bool {
	var req *http.Request
	var buffReader io.Reader

	if d.config.Limit != 0 && d.ammoNum >= d.config.Limit {
		d.err = ErrAmmoLimit
		return false
	}
	for {
		select {
		case <-ctx.Done():
			d.err = ctx.Err()
			return false
		default:
		}
		data, err := d.reader.ReadString('\n')
		if err == io.EOF {
			d.passNum++
			if d.config.Passes != 0 && d.passNum >= d.config.Passes {
				d.err = ErrPassLimit
				return false
			}
			if d.ammoNum == 0 {
				d.err = ErrNoAmmo
				return false
			}
			d.header = make(http.Header)
			_, err := d.file.Seek(0, io.SeekStart)
			if err != nil {
				d.err = err
				return false
			}
			d.reader.Reset(d.file)
			continue
		}
		if err != nil {
			d.err = xerrors.Errorf("reading ammo failed at position: %d, err: %w", filePosition(d.file), err)
			return false
		}
		data = strings.TrimSpace(data)
		if len(data) == 0 {
			continue // skip empty lines
		}
		if data[0] == '[' {
			key, val, err := util.DecodeHeader(string(data))
			if err == nil {
				d.header.Set(key, val)
			}
			continue
		}
		d.ammoNum++
		bodySize, uri, tag, err := uripost.DecodeURI(data)
		if err != nil {
			d.err = err
			return false
		}

		buffReader = nil
		buff := make([]byte, bodySize)
		if bodySize != 0 {
			if n, err := io.ReadFull(d.reader, buff); err != nil {
				d.err = xerrors.Errorf("failed to read ammo with err: %w, at position: %v; tried to read: %v; have read: %v", err, filePosition(d.file), bodySize, n)
				return false
			}
			buffReader = bytes.NewReader(buff)
		}
		req, err = http.NewRequest("POST", uri, buffReader)
		if err != nil {
			d.err = xerrors.Errorf("failed to decode ammo with err: %w, at position: %v; data: %q", err, filePosition(d.file), buff)
		}

		for k, v := range d.header {
			// http.Request.Write sends Host header based on req.URL.Host
			if k == "Host" {
				req.Host = v[0]
			} else {
				req.Header[k] = v
			}
		}

		// add new Headers to request from config
		util.EnrichRequestWithHeaders(req, d.decodedConfigHeaders)
		d.req = req
		d.tag = tag
		return true
	}
}
