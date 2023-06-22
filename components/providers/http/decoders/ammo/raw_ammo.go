package ammo

import (
	"net/http"

	"github.com/yandex/pandora/components/providers/http/decoders/raw"
	"github.com/yandex/pandora/components/providers/http/util"
	"golang.org/x/xerrors"
)

type RawAmmo struct {
	buff          []byte
	filePosition  int64
	tag           string
	commonHeaders http.Header
}

func (a *RawAmmo) BuildRequest() (*http.Request, error) {
	req, err := raw.DecodeRequest(a.buff)
	if err != nil {
		return nil, xerrors.Errorf("failed to decode ammo with err: %w, at position: %v; data: %q", err, a.filePosition, a.buff)
	}
	util.EnrichRequestWithHeaders(req, a.commonHeaders)
	return req, nil
}

func (a *RawAmmo) Tag() string {
	return a.tag
}

func (a *RawAmmo) Setup(buff []byte, tag string, filePosition int64, header http.Header) {
	a.buff = buff
	a.tag = tag
	a.filePosition = filePosition
	a.commonHeaders = header.Clone()
}

func (a *RawAmmo) Reset() {
	a.buff = nil
	a.filePosition = 0
	a.tag = ""
	a.commonHeaders = nil
}
