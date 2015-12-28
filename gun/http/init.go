package http

import (
	"errors"
	"fmt"

	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/gun"
)

func New(c *config.Gun) (gun.Gun, error) {
	params := c.Parameters
	if params == nil {
		return nil, errors.New("Parameters not specified")
	}
	target, ok := params["Target"]
	if !ok {
		return nil, errors.New("Target not specified")
	}
	g := &FastHttpGun{}
	switch t := target.(type) {
	case string:
		g.target = target.(string)
	default:
		return nil, fmt.Errorf("Target is of the wrong type."+
			" Expected 'string' got '%T'", t)
	}
	if ssl, ok := params["SSL"]; ok {
		if sslVal, casted := ssl.(bool); casted {
			g.ssl = sslVal
		} else {
			return nil, fmt.Errorf("SSL should be boolean type.")
		}
	}
	return g, nil
}
