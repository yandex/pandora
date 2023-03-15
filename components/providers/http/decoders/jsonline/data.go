//go:generate ffjson $GOFILE

package jsonline

import (
	"net/http"
	"strings"

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

func (d *data) ToRequest() (*http.Request, error) {
	uri := "http://" + d.Host + d.URI
	req, err := http.NewRequest(d.Method, uri, strings.NewReader(d.Body))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	for k, v := range d.Headers {
		req.Header.Set(k, v)
	}
	return req, err
}

func DecodeAmmo(jsonDoc []byte) (*http.Request, string, error) {
	var data = new(data)
	if err := data.UnmarshalJSON(jsonDoc); err != nil {
		err = errors.WithStack(err)
		return nil, data.Tag, err
	}
	req, err := data.ToRequest()
	return req, data.Tag, err
}
