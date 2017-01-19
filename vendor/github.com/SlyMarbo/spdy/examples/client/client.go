// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"

	_ "github.com/SlyMarbo/spdy" // This adds SPDY support to net/http
)

func main() {
	res, err := http.Get("https://example.com/")
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Received: %s\n", bytes)
}
