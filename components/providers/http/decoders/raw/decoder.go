package raw

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

func DecodeHeader(headerString string) (reqSize int, tag string, err error) {
	var sizeStr string
	sizeStr, tag, _ = strings.Cut(headerString, " ")
	reqSize, err = strconv.Atoi(sizeStr)
	if err != nil {
		return 0, "", fmt.Errorf("invalid payload size line `%s`. expect `%%d %%s`", headerString)
	}
	return reqSize, tag, err
}

func DecodeRequest(reqString []byte) (req *http.Request, err error) {
	reader := bufio.NewReader(bytes.NewReader(reqString))
	req, err = http.ReadRequest(reader)
	if err != nil {
		return
	}
	req.RequestURI = ""

	if req.Body == http.NoBody && req.Header.Get("Content-Length") == "" {
		rest, readErr := io.ReadAll(reader)
		if readErr != nil {
			return nil, readErr
		}
		if len(rest) > 0 {
			req.Body = io.NopCloser(bytes.NewReader(rest))
			req.GetBody = func() (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(rest)), nil
			}
		}
	}

	return req, err
}
