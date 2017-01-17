// Copyright 2014 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spdy_test

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/SlyMarbo/spdy/common"
)

func issue82handler(n int64) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write(bytes.Repeat([]byte{'X'}, int(n)))
	})
}

func TestIssue82(t *testing.T) {
	var max int64 = common.DEFAULT_INITIAL_CLIENT_WINDOW_SIZE
	tests := []struct {
		Name string
		Path string
		Msg  string
		Want int64
	}{
		{"Issue 82", "/1", "Sending less than window size", max - 16},
		{"Issue 82", "/2", "Sending just under window size", max - 15},
		{"Issue 82", "/3", "Sending exactly window size", max},
		{"Issue 82", "/4", "Sending more than window size", max + 16},
	}

	// start server
	mux := http.NewServeMux()
	for _, test := range tests {
		mux.HandleFunc(test.Path, issue82handler(test.Want))
	}

	srv := newServer(mux)
	defer srv.Close()

	client := newClient()

	for _, test := range tests {
		r, err := client.Get(srv.URL + test.Path)
		if err != nil {
			t.Fatal(err)
		}

		n, err := io.Copy(ioutil.Discard, r.Body)
		if err != nil {
			t.Fatal(err)
		}

		if err = r.Body.Close(); err != nil {
			t.Fatal(err)
		}

		if n != test.Want {
			t.Errorf("%s: %s, got %d, expected %d", test.Name, test.Msg, n, test.Want)
		}
	}
}
