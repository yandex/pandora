package main

import (
	"fmt"
	"github.com/amahi/spdy"
	"net/http"
)

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func main() {
        spdy.EnableDebug()
        http.HandleFunc("/", handler)
	err := spdy.ListenAndServeTLS("localhost:4040", "server.pem", "server.key" , nil)
	if err != nil {
		fmt.Println(err)
	}
}
