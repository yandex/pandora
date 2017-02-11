//go:generate ffjson $GOFILE

package jsonline

// ffjson: noencoder
type data struct {
	Host    string            `json:"host"`
	Method  string            `json:"method"`
	Uri     string            `json:"uri"`
	Headers map[string]string `json:"headers"`
	Tag     string            `json:"tag"`
}
