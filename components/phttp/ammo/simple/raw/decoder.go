package raw

import (
	"net/http"
	"bytes"
	"bufio"
)

func Decode(reqString []byte) (req *http.Request, err error) {
	return http.ReadRequest(bufio.NewReader(bytes.NewReader(reqString)))
}
