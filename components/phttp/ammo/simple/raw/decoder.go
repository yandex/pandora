package raw

import (
	"bufio"
	"bytes"
	"net/http"
	"strconv"
	"strings"
)

func DecodeHeader(headerString []byte) (reqSize int, tag string, err error) {
	parts := strings.SplitN(string(headerString), " ", 2)
	reqSize, err = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		tag = parts[1]
	} else {
		tag = "__EMPTY__"
	}
	return
}

func DecodeRequest(reqString []byte) (req *http.Request, err error) {
	return http.ReadRequest(bufio.NewReader(bytes.NewReader(reqString)))
}
