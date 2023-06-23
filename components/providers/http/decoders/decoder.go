package decoders

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/util"
	"github.com/yandex/pandora/core"
)

//go:generate go run github.com/vektra/mockery/v2@v2.22.1 --inpackage --name=Decoder --filename=mock_decoder.go

func filePosition(file io.ReadSeeker) (position int64) {
	position, _ = file.Seek(0, io.SeekCurrent)
	return
}

var (
	ErrUnknown   = fmt.Errorf("unknown decoder faced")
	ErrNoAmmo    = fmt.Errorf("no ammo in file")
	ErrAmmoLimit = fmt.Errorf("ammo limit faced")
	ErrPassLimit = fmt.Errorf("passes limit faced")
)

type Decoder interface {
	Scan(context.Context) (DecodedAmmo, error)
	Release(a core.Ammo)
	LoadAmmo(context.Context) ([]DecodedAmmo, error)
}

type protoDecoder struct {
	file                 io.ReadSeeker
	config               config.Config
	decodedConfigHeaders http.Header
	ammoNum              uint // number of ammo reads
	passNum              uint // number of file reads
}

func (d *protoDecoder) LoadAmmo(ctx context.Context, scan func(ctx context.Context) (DecodedAmmo, error)) ([]DecodedAmmo, error) {
	passes := d.config.Passes
	limit := d.config.Limit
	d.config.Passes = 1
	d.config.Limit = 0
	var result []DecodedAmmo
	var err error
	var ammo DecodedAmmo
	for err == nil {
		ammo, err = scan(ctx)
		if ammo != nil {
			result = append(result, ammo)
		}
	}
	d.config.Passes = passes
	d.config.Limit = limit
	if errors.Is(err, ErrPassLimit) {
		err = nil
	}

	return result, err
}

func NewDecoder(conf config.Config, file io.ReadSeeker) (d Decoder, err error) {
	decodedConfigHeaders, err := util.DecodeHTTPConfigHeaders(conf.Headers)
	if err != nil {
		return
	}

	switch conf.Decoder {
	case config.DecoderJSONLine:
		d = newJsonlineDecoder(file, conf, decodedConfigHeaders)
	case config.DecoderRaw:
		d = newRawDecoder(file, conf, decodedConfigHeaders)
	case config.DecoderURI:
		d = newURIDecoder(file, conf, decodedConfigHeaders)
	case config.DecoderURIPost:
		d = newURIPostDecoder(file, conf, decodedConfigHeaders)
	default:
		err = ErrUnknown
	}
	return
}
