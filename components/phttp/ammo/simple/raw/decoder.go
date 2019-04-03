package raw

import (
	"bufio"
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

func decodeHeader(headerString []byte) (reqSize int, tag string, err error) {
	parts := strings.SplitN(string(headerString), " ", 2)
	reqSize, err = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		tag = parts[1]
	}
	return
}

func decodeRequest(reqString []byte) (req *http.Request, err error) {
	req, err = http.ReadRequest(bufio.NewReader(bytes.NewReader(reqString)))
	if err != nil {
		return
	}
	if req.Host != "" {
		req.URL.Host = req.Host
	}
	req.RequestURI = ""
	return
}

func decodeConfigHeader(req *http.Request, line []byte) error {
	if len(line) < 3 || line[0] != '[' || line[len(line)-1] != ']' {
		return errors.New("header line should be like '[key: value]")
	}
	line = line[1 : len(line)-1]
	colonIdx := bytes.IndexByte(line, ':')
	if colonIdx < 0 {
		return errors.New("missing colon")
	}
	key := string(bytes.TrimSpace(line[:colonIdx]))
	val := string(bytes.TrimSpace(line[colonIdx+1:]))
	if key == "" {
		return errors.New("missing header key")
	}
	if strings.ToLower(key) == "host" {
		req.URL.Host = val
	}
	req.Header.Set(key, val)
	return nil
}
