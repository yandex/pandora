//go:generate github.com/pquerna/ffjson@latest data_ffjson.go

package jsonline

import (
	"net/http"

	"github.com/pkg/errors"
	"github.com/yandex/pandora/components/providers/base"
)

// ffjson: noencoder
type data struct {
	// Host defines Host header to send.
	// Request endpoint is defied by gun config.
	Host   string `json:"host"`
	Method string `json:"method"`
	URI    string `json:"uri"`
	// Headers defines headers to send.
	// NOTE: Host header will be silently ignored.
	Headers map[string]string `json:"headers"`
	Tag     string            `json:"tag"`
	// Body should be string, doublequotes should be escaped for json body
	Body string `json:"body"`
}

func DecodeAmmo(jsonDoc []byte, headers http.Header) (*base.Ammo, error) {
	var d = new(data)
	if err := d.UnmarshalJSON(jsonDoc); err != nil {
		err = errors.WithStack(err)
		return nil, err
	}

	for k, v := range d.Headers {
		headers.Set(k, v)
	}
	url := "http://" + d.Host + d.URI
	var body []byte
	if d.Body != "" {
		body = []byte(d.Body)
	}
	return base.NewAmmo(d.Method, url, body, headers, d.Tag)
}
