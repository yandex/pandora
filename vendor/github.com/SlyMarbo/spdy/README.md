# Deprecated

With the release of [Go1.6](https://golang.org/doc/go1.6) and the addition of [http2](https://golang.org/x/net/http2)
to the standard library, this package is no longer under active development. It is highly recommended that former
users of this package migrate to HTTP/2.

# spdy

[![GoDoc](https://godoc.org/github.com/SlyMarbo/spdy?status.png)](https://godoc.org/github.com/SlyMarbo/spdy)

A full-featured SPDY library for the Go language.
 
Note that this implementation currently supports SPDY drafts 2 and 3.

See [these examples][examples] for a quick intro to the package.

[examples]: https://github.com/SlyMarbo/spdy/tree/master/examples

Note that using this package with [Martini][martini] is likely to result in strange and hard-to-diagnose
bugs. For more information, read [this article][martini-article]. As a result, issues that arise when
combining the two should be directed at the Martini developers.

[martini]: https://github.com/go-martini/martini
[martini-article]: http://stephensearles.com/?p=254

Servers
-------


The following examples use features specific to SPDY.

Just the handler is shown.

Use SPDY's pinging features to test the connection:
```go
package main

import (
	"net/http"
	"time"

	"github.com/SlyMarbo/spdy"
)

func Serve(w http.ResponseWriter, r *http.Request) {
	// Ping returns a channel which will send an empty struct.
	if ping, err := spdy.PingClient(w); err == nil {
		select {
		case response := <- ping:
			if response != nil {
				// Connection is fine.
			} else {
				// Something went wrong.
			}
			
		case <-time.After(timeout):
			// Ping took too long.
		}
	} else {
		// Not SPDY
	}
	
	// ...
}
```



Sending a server push:
```go
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
```
