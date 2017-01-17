// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/SlyMarbo/spdy"
)

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

func handleProxy(conn spdy.Conn) {
	url := "http://" + conn.Conn().RemoteAddr().String() + "/"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	res, err := conn.RequestResponse(req, nil, 1)
	if err != nil {
		panic(err)
	}

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, res.Body)
	if err != nil {
		panic(err)
	}

	res.Body.Close()

	fmt.Println(buf.String())
}

func main() {
	handler := spdy.ProxyConnHandlerFunc(handleProxy)
	http.Handle("/", spdy.ProxyConnections(handler))
	handle(http.ListenAndServeTLS(":8080", "cert.pem", "key.pem", nil))
}
