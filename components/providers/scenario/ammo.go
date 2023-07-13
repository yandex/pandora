package scenario

import (
	"net/http"
)

type InputParam string

type OutputParams string

type Ammo struct {
	InputParams  []InputParam
	OutputParams []OutputParams
}

func NewAmmo(method string, url string, body []byte, header http.Header, tag string) (*Ammo, error) {
	// Base, err := base.NewAmmo(method, url, body, header, tag)

	// if err != nil {
	// 	return nil, err
	// }
	return &Ammo{
		// Ammo: *Base,
	}, nil
}

func (a *Ammo) Reset() {
	a.InputParams = []InputParam{}
	a.OutputParams = []OutputParams{}
}
