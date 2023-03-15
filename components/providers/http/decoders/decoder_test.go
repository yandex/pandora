package decoders

import "net/http"

type DecodeInput struct{}
type DecodeWant struct {
	req *http.Request
}
