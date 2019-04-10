package raw

import (
	"bufio"
	"bytes"
	"net/http"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

type Header struct {
	key   string
	value string
}

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

func decodeHTTPConfigHeaders(headers []string) (configHTTPHeaders []Header, err error) {
	for _, header := range headers {
		line := []byte(header)
		if len(line) < 3 || line[0] != '[' || line[len(line)-1] != ']' {
			return nil, errors.New("header line should be like '[key: value]")
		}
		line = line[1 : len(line)-1]
		colonIdx := bytes.IndexByte(line, ':')
		if colonIdx < 0 {
			return nil, errors.New("missing colon")
		}
		configHTTPHeaders = append(
			configHTTPHeaders,
			Header{
				string(bytes.TrimSpace(line[:colonIdx])),
				string(bytes.TrimSpace(line[colonIdx+1:])),
			})
	}
	return
}
