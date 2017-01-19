// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// Miscellaneous functions

package spdy

import (
	"fmt"
	"net"
	"net/url"
	"syscall"
)

// PriorityFor returns the recommended priority for the given URL
// for best opteration with the library.
func PriorityFor(req *url.URL) uint8 {
	// FIXME: need to implement priorities properly
	return 4
}

// check to see if err is a connection reset
func isConnReset(err error) bool {
	if e, ok := err.(*net.OpError); ok {
		if errno, ok := e.Err.(syscall.Errno); ok {
			return errno == syscall.ECONNRESET
		}
	}
	return false
}

// check to see if err is an network timeout
func isBrokenPipe(err error) bool {
	if e, ok := err.(*net.OpError); ok {
		return e.Err == syscall.EPIPE
	}
	return false
}

// return best guess at a string for a network error
func netErrorString(err error) string {
	if e, ok := err.(*net.OpError); ok {
		return fmt.Sprintf("%s", e.Err)
	}
	return fmt.Sprintf("%#v", err)
}
