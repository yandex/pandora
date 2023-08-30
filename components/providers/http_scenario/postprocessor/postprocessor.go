package postprocessor

import (
	"io"
	"net/http"
)

type Config struct {
	Mapping map[string]string
}

type Postprocessor interface {
	Process(reqMap map[string]any, resp *http.Response, body io.Reader) error
}
