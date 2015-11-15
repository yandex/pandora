package engine_test

import (
	"os"
	"flag"
	"testing"
	"net/http"
	"time"
	"log"

	"github.com/amahi/spdy"
)

func runSpdyTestServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("This is an example server.\n"))
	})

	//use spdy's Listen and serve
	log.Println("Run spdy server on localhost:3000")
	err := spdy.ListenAndServeTLS("localhost:3000",
		"./testdata/test.crt", "./testdata/test.key", nil)
	if err != nil {
		//error handling here
		log.Panic(err)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	go runSpdyTestServer()
	time.Sleep(time.Millisecond * 5) // wait for server
	os.Exit(m.Run())
}