package decoders

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/yandex/pandora/components/providers/base"
	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/util"
)

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
	// Decode(context.Context, chan<- *base.Ammo[http.Request], io.ReadSeeker) error
	Scan(context.Context) (*base.Ammo, error)
}

type protoDecoder struct {
	file                 io.ReadSeeker
	config               config.Config
	decodedConfigHeaders http.Header
	ammoNum              uint // number of ammo reads
	passNum              uint // number of file reads
}

func NewDecoder(conf config.Config, file io.ReadSeeker) (d Decoder, err error) {
	var decodedConfigHeaders http.Header
	decodedConfigHeaders, err = util.DecodeHTTPConfigHeaders(conf.Headers)
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
