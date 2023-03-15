package decoders

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"

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
	Scan(context.Context) bool
	Next() (*http.Request, string)
	Err() error
}

type protoDecoder struct {
	file io.ReadSeeker
	config.Config
	decodedConfigHeaders http.Header
	ammoNum              uint
	passNum              uint

	req *http.Request
	tag string
	err error
}

func (d *protoDecoder) Next() (*http.Request, string) {
	return d.req, d.tag
}

func (d *protoDecoder) Err() error {
	return d.err
}

type Request interface {
	http.Request
}

func NewDecoder(conf config.Config, file io.ReadSeeker) (d Decoder, err error) {
	var decodedConfigHeaders http.Header
	decodedConfigHeaders, err = util.DecodeHTTPConfigHeaders(conf.Headers)
	if err != nil {
		return
	}

	proto := protoDecoder{
		file:                 file,
		Config:               conf,
		decodedConfigHeaders: decodedConfigHeaders,
	}

	switch conf.Decoder {
	case config.DecoderJSONLine:
		scanner := bufio.NewScanner(file)
		if conf.MaxAmmoSize != 0 {
			var buffer []byte
			scanner.Buffer(buffer, conf.MaxAmmoSize)
		}
		d = &jsonlineDecoder{protoDecoder: proto, scanner: scanner}
	case config.DecoderRaw:
		d = &rawDecoder{protoDecoder: proto, reader: bufio.NewReader(file)}
	case config.DecoderURI:
		d = &uriDecoder{protoDecoder: proto, scanner: bufio.NewScanner(file), Header: make(http.Header)}
	case config.DecoderURIPost:
		d = &uripostDecoder{protoDecoder: proto, reader: bufio.NewReader(file), header: make(http.Header)}
	default:
		err = ErrUnknown
	}
	return
}
