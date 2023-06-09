package base

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	urlpkg "net/url"

	"github.com/yandex/pandora/components/providers/http/util"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/lib/netutil"
)

func NewAmmo(method string, url string, body []byte, header http.Header, tag string) (*Ammo, error) {
	if ok := netutil.ValidHTTPMethod(method); !ok {
		return nil, errors.New("invalid HTTP method " + method)
	}
	if _, err := urlpkg.Parse(url); err != nil {
		return nil, fmt.Errorf("invalid URL %s; err %w ", url, err)
	}
	return &Ammo{
		method:      method,
		body:        body,
		url:         url,
		tag:         tag,
		header:      header,
		constructor: true,
	}, nil
}

type Ammo struct {
	Req         *http.Request
	method      string
	body        []byte
	url         string
	tag         string
	header      http.Header
	id          uint64
	isInvalid   bool
	constructor bool
}

func (a *Ammo) Request() (*http.Request, *netsample.Sample) {
	if a.Req == nil {
		_ = a.BuildRequest() // TODO: what if error. There isn't a logger
	}
	sample := netsample.Acquire(a.Tag())
	sample.SetID(a.ID())
	return a.Req, sample
}

func (a *Ammo) SetID(id uint64) {
	a.id = id
}

func (a *Ammo) ID() uint64 {
	return a.id
}

func (a *Ammo) Invalidate() {
	a.isInvalid = true
}

func (a *Ammo) IsInvalid() bool {
	return a.isInvalid
}

func (a *Ammo) IsValid() bool {
	return !a.isInvalid
}

func (a *Ammo) SetTag(tag string) {
	a.tag = tag
}

func (a *Ammo) Tag() string {
	return a.tag
}

func (a *Ammo) FromConstructor() bool {
	return a.constructor
}

// use NewAmmo() for skipping error here
func (a *Ammo) BuildRequest() error {
	var buff io.Reader
	if a.body != nil {
		buff = bytes.NewReader(a.body)
	}
	req, err := http.NewRequest(a.method, a.url, buff)
	if err != nil {
		return fmt.Errorf("cant create request: %w", err)
	}
	a.Req = req
	util.EnrichRequestWithHeaders(req, a.header)
	return nil
}

func (a *Ammo) Reset() {
	a.Req = nil
	a.method = ""
	a.body = nil
	a.url = ""
	a.tag = ""
	a.header = nil
	a.id = 0
	a.isInvalid = false
}
