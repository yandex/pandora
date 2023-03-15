package uripost

import (
	"fmt"
	"strconv"
	"strings"
)

var ErrWrongSize = fmt.Errorf("wrong ammo bodySize format: should be int, in 'bodySize uri [tag]'")
var ErrAmmoFormat = fmt.Errorf("wrong ammo format: should be like 'bodySize uri [tag]'")

func DecodeURI(uriString string) (bodySize int, uri string, tag string, err error) {
	parts := strings.Split(uriString, " ")
	if len(parts) < 2 {
		err = ErrAmmoFormat
	} else {
		bodySize, err = strconv.Atoi(parts[0])
		if err != nil {
			err = ErrWrongSize
			return
		}
		uri = parts[1]
		if len(parts) > 2 {
			tag = strings.Join(parts[2:], " ")
		}
	}

	return
}
