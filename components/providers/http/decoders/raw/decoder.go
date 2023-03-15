package raw

import (
	"bufio"
	"bytes"
	"net/http"
	"strconv"
	"strings"
)

func DecodeHeader(headerString string) (reqSize int, tag string, err error) {
	var sizeStr string
	sizeStr, tag, _ = strings.Cut(headerString, " ")
	reqSize, err = strconv.Atoi(sizeStr)
	return reqSize, tag, err
}

func DecodeRequest(reqString []byte) (req *http.Request, err error) {
	req, err = http.ReadRequest(bufio.NewReader(bytes.NewReader(reqString)))
	if err != nil {
		return
	}
	req.RequestURI = ""
	return req, err
}
