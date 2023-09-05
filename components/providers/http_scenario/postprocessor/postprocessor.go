package postprocessor

import (
	"io"
	"net/http"
)

type Config struct {
	Mapping map[string]string
}

type Postprocessor interface {
	Process(resp *http.Response, body io.Reader) (map[string]any, error)
}
