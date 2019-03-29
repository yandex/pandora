package raw

import (
	"bufio"
	"bytes"
	"net/http"
	"strconv"
	"strings"
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
