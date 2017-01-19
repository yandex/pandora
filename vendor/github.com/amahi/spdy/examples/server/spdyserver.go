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
	http.HandleFunc("/", handler)
	err := spdy.ListenAndServe("localhost:4040",nil)
	if err != nil {
		fmt.Println(err)
	}
}

