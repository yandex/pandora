package scenario

import (
	"io"
)

type decoder struct {
}

func (d decoder) parseAmmo(file io.ReadSeeker, conf Config) ([]*Ammo, error) {

	return nil, nil
}
