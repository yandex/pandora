package decoders

import "net/http"

type DecodedAmmo interface {
	BuildRequest() (*http.Request, error)
	Tag() string
}
