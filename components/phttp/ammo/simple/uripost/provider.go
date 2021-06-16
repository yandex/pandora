package uripost

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"go.uber.org/zap"

	"github.com/yandex/pandora/components/phttp/ammo/simple"
)

func filePosition(file afero.File) (position int64) {
	position, _ = file.Seek(0, io.SeekCurrent)
	return
}

type Config struct {
	File string `validate:"required"`
	// Limit limits total num of ammo. Unlimited if zero.
	Limit int `validate:"min=0"`
	// Redefine HTTP headers
	Headers []string
	// Passes limits ammo file passes. Unlimited if zero.
	Passes int `validate:"min=0"`
}

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
	log *zap.Logger
}

func (p *Provider) start(ctx context.Context, ammoFile afero.File) error {
	var passNum int
	var ammoNum int
	//	var key string
	//	var val string
	var bodySize int
	var uri string
	var tag string

	header := make(http.Header)
	// parse and prepare Headers from config
	decodedConfigHeaders, err := decodeHTTPConfigHeaders(p.Config.Headers)
	if err != nil {
		return err
	}
	for {
		passNum++
		reader := bufio.NewReader(ammoFile)
		for p.Limit == 0 || ammoNum < p.Limit {
			data, isPrefix, err := reader.ReadLine()
			if isPrefix {
				return errors.Errorf("too long header in ammo at position %v", filePosition(ammoFile))
			}
			if err == io.EOF {
				break // start over from the beginning
			}
			if err != nil {
				return errors.Wrapf(err, "reading ammo failed at position: %v", filePosition(ammoFile))
			}
			if len(data) == 0 {
				continue // skip empty lines
			}
			data = bytes.TrimSpace(data)
			if data[0] == '[' {
				key, val, err := decodeHeader(data)
				if err == nil {
					header.Set(key, val)
				}
				continue
			}
			if _, err := strconv.Atoi(string(data[0])); err == nil {
				bodySize, uri, tag, _ = decodeURI(data)
			}
			if bodySize == 0 {
				break // start over from the beginning of file if ammo size is 0
			}
			buff := make([]byte, bodySize)
			if n, err := io.ReadFull(reader, buff); err != nil {
				return errors.Wrapf(err, "failed to read ammo at position: %v; tried to read: %v; have read: %v", filePosition(ammoFile), bodySize, n)
			}
			req, err := http.NewRequest("POST", uri, bytes.NewReader(buff))
			if err != nil {
				return errors.Wrapf(err, "failed to decode ammo at position: %v; data: %q", filePosition(ammoFile), buff)
			}

			for k, v := range header {
				// http.Request.Write sends Host header based on req.URL.Host
				if k == "Host" {
					req.Host = v[0]
					req.URL.Host = v[0]
				} else {
					req.Header[k] = v
				}
			}

			// redefine request Headers from config
			for _, header := range decodedConfigHeaders {
				// special behavior for `Host` header
				if header.key == "Host" {
					req.URL.Host = header.value
				} else {
					req.Header.Set(header.key, header.value)
				}
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
			return errors.New("no ammo in file")
		}
		if p.Passes != 0 && passNum >= p.Passes {
			break
		}
		_, err := ammoFile.Seek(0, 0)
		if err != nil {
			p.log.Info("Failed to seek ammo file", zap.Error(err))
		}
	}
	return nil
}
