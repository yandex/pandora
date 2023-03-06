package uripost

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func decodeHeader(line []byte) (key string, val string, err error) {
	if len(line) < 3 || line[0] != '[' || line[len(line)-1] != ']' {
		return key, val, errors.New("header line should be like '[key: value]'")
	}
	line = line[1 : len(line)-1]
	colonIdx := bytes.IndexByte(line, ':')
	if colonIdx < 0 {
		return key, val, errors.New("missing colon")
	}
	key = string(bytes.TrimSpace(line[:colonIdx]))
	val = string(bytes.TrimSpace(line[colonIdx+1:]))
	if key == "" {
		return key, val, errors.New("missing header key")
	}
	return
}

func decodeURI(uriString []byte) (bodySize int, uri string, tag string, err error) {
	parts := strings.Split(string(uriString), " ")
	bodySize, err = strconv.Atoi(parts[0])
	if err != nil {
		err = errors.New("Wrong ammo body size, should be in bytes")
		return
	}
	switch {
	case len(parts) == 2:
		uri = parts[1]
	case len(parts) >= 3:
		uri = parts[1]
		tag = parts[2]
	default:
		err = errors.New("Wrong ammo format, should be like 'bodySize uri [tag]'")
	}

	return
}
