// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// Proxy function

package spdy

import (
	"net/http"
)

// NewStreamProxy starts a new stream and proxies the given HTTP Request to
// it, writing the response to the given ResponseWriter. If there is an error,
// it will be returned, but the ResponseWriter will get a 404 Not Found.
func (s *Session) NewStreamProxy(r *http.Request, w http.ResponseWriter) (err error) {

	str := s.NewClientStream()
	if str == nil {
		log.Println("ERROR in NewClientStream: cannot create stream")
		http.NotFound(w, r)
		return
	}
	err = str.Request(r, w)
	if err != nil {
		http.NotFound(w, r)
		log.Println("ERROR in Request:", err)
		return
	}

	return
}
