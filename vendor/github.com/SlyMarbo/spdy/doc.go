// Copyright 2013 Jamie Hall. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package spdy is a full-featured SPDY library for the Go language (still under very active development).

Note that this implementation currently supports SPDY drafts 2 and 3, and support for SPDY/4, and HTTP/2.0 is upcoming.

See examples for various simple examples that use the package.

-------------------------------

Note that using this package with Martini (https://github.com/go-martini/martini) is likely to result
in strange and hard-to-diagnose bugs. For more information, read http://stephensearles.com/?p=254.
As a result, issues that arise when combining the two should be directed at the Martini developers.

-------------------------------

		Servers

The following examples use features specific to SPDY.

Just the handler is shown.

Use SPDY's pinging features to test the connection:

		package main

		import (
			"net/http"
			"time"

			"github.com/SlyMarbo/spdy"
		)

		func Serve(w http.ResponseWriter, r *http.Request) {
			// Ping returns a channel which will send a bool.
			if ping, err := spdy.PingClient(w); err == nil {
				select {
				case _, ok := <- ping:
					if ok {
						// Connection is fine.
					} else {
						// Something went wrong.
					}

				case <-time.After(timeout):
					// Ping took too long.
				}
			} else {
				// Not SPDY.
			}

			// ...
		}


Sending a server push:

		package main

		import (
			"net/http"

			"github.com/SlyMarbo/spdy"
		)

		func Serve(w http.ResponseWriter, r *http.Request) {
			// Push returns a separate http.ResponseWriter and an error.
			path := r.URL.Scheme + "://" + r.URL.Host + "/example.js"
			push, err := spdy.Push(path)
			if err != nil {
				// Not using SPDY.
			}
			http.ServeFile(push, r, "./content/example.js")

			// Note that a PushStream must be finished manually once
			// all writing has finished.
			push.Finish()

			// ...
		}

*/
package spdy
