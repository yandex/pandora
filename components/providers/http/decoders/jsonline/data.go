//go:generate github.com/pquerna/ffjson@latest data_ffjson.go

package jsonline

import (
	"net/http"

	"github.com/pkg/errors"
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

func DecodeAmmo(jsonDoc []byte, baseHeader http.Header) (method string, url string, header http.Header, tag string, body []byte, err error) {
	var d = new(data)
	if err := d.UnmarshalJSON(jsonDoc); err != nil {
		err = errors.WithStack(err)
		return "", "", nil, "", nil, err
	}

	header = baseHeader.Clone()
	for k, v := range d.Headers {
		header.Set(k, v)
	}
	url = "http://" + d.Host + d.URI
	if d.Body != "" {
		body = []byte(d.Body)
	}
	return d.Method, url, header, d.Tag, body, nil
}
