package ammo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	url2 "net/url"

	"github.com/yandex/pandora/components/providers/http/util"
	"github.com/yandex/pandora/lib/netutil"
)

type Ammo struct {
	method string
	body   []byte
	url    string
	tag    string
	header http.Header
}

func (a *Ammo) BuildRequest() (*http.Request, error) {
	var buff io.Reader
	if a.body != nil {
		buff = bytes.NewReader(a.body)
	}
	req, err := http.NewRequest(a.method, a.url, buff)
	if err != nil {
		return nil, fmt.Errorf("cant create request: %w", err)
	}
	util.EnrichRequestWithHeaders(req, a.header)
	return req, nil
}

func (a *Ammo) Tag() string {
	return a.tag
}

func (a *Ammo) Setup(method string, url string, body []byte, header http.Header, tag string) error {
	if ok := netutil.ValidHTTPMethod(method); !ok {
		return errors.New("invalid HTTP method " + method)
	}
	if _, err := url2.Parse(url); err != nil {
		return fmt.Errorf("invalid URL %s; err %w ", url, err)
	}

	a.method = method
	a.body = body
	a.url = url
	a.tag = tag
	a.header = header
	return nil
}

func (a *Ammo) Reset() {
	a.method = ""
	a.body = nil
	a.url = ""
	a.tag = ""
	a.header = nil
}
