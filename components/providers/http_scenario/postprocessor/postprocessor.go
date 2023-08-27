package postprocessor

import "net/http"

type Config struct {
	Mapping map[string]string
}

type Postprocessor interface {
	Process(reqMap map[string]any, resp *http.Response, body []byte) error
}
