//go:generate ffjson $GOFILE

package jsonline

// ffjson: noencoder
type data struct {
	// Host defines Host header to send.
	// Request endpoint is defied by gun config.
	Host   string `json:"host"`
	Method string `json:"method"`
	Uri    string `json:"uri"`
	// Headers defines headers to send.
	// NOTE: Host header will be silently ignored.
	Headers map[string]string `json:"headers"`
	Tag     string            `json:"tag"`
	// Body should be string, doublequotes should be escaped for json body
	Body string `json:"body"`
}
