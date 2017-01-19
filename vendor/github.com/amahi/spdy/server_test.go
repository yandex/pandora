// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// friendly API tests

package spdy

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

const SERVER_CERTFILE = "cert/serverTLS/server.pem"
const SERVER_KEYFILE = "cert/serverTLS/server.key"
const CLIENT_CERTFILE = "cert/clientTLS/client.pem"
const CLIENT_KEYFILE = "cert/clientTLS/client.key"

func init() {
	SetLog(ioutil.Discard)
}

//handler for requests
func ServerHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func StartServer(server *Server, done chan<- error) {
	err := server.ListenAndServe()
	if err != nil {
		done <- err
	}
	done <- errors.New("")
}

func testclient(done chan<- error) {
	//make client
	client, err := NewClient("localhost:4040")
	if err != nil {
		done <- err
	}
	for i := 0; i < 100; i++ {
		//now send requests and test
		req, err := http.NewRequest("GET", "http://localhost:4040/banana", nil)
		if err != nil {
			done <- err
		}
		res, err := client.Do(req)
		if err != nil {
			done <- err
		}
		data := make([]byte, int(res.ContentLength))
		_, err = res.Body.(io.Reader).Read(data)
		if string(data) != "Hi there, I love banana!" {
			done <- err
		}
		res.Body.Close()

		//another request
		req, err = http.NewRequest("POST", "http://localhost:4040/monkeys", bytes.NewBufferString("hello=world"))
		if err != nil {
			done <- err
		}

		res, err = client.Do(req)
		if err != nil {
			done <- err
		}
		data = make([]byte, int(res.ContentLength))
		_, err = res.Body.(io.Reader).Read(data)
		if string(data) != "Hi there, I love monkeys!" {
			done <- err
		}
		res.Body.Close()
	}

	//close client
	err = client.Close()
	if err != nil {
		done <- err
	}
	done <- errors.New("")
}

//simple server with many clients sending requests to it
func TestSimpleServerClient(t *testing.T) {
	//make server
	mux := http.NewServeMux()
	mux.HandleFunc("/", ServerHandler)
	server := &Server{
		Addr:    "localhost:4040",
		Handler: mux,
	}
	serverdone := make(chan error)
	go StartServer(server, serverdone)
	time.Sleep(100 * time.Millisecond)

	cldone1 := make(chan error)
	cldone2 := make(chan error)
	cldone3 := make(chan error)
	cldone4 := make(chan error)
	cldone5 := make(chan error)

	go testclient(cldone1)
	go testclient(cldone2)
	go testclient(cldone3)
	go testclient(cldone4)
	go testclient(cldone5)

	err := <-cldone1
	if err.Error() != "" {
		t.Fatal(err.Error())
	}
	err = <-cldone2
	if err.Error() != "" {
		t.Fatal(err.Error())
	}
	err = <-cldone3
	if err.Error() != "" {
		t.Fatal(err.Error())
	}
	err = <-cldone4
	if err.Error() != "" {
		t.Fatal(err.Error())
	}
	err = <-cldone5
	if err.Error() != "" {
		t.Fatal(err.Error())
	}

	server.Close()
	time.Sleep(100 * time.Millisecond)
	err = <-serverdone
}

func TestTLSServerSpdyOnly(t *testing.T) {
	//make server
	mux := http.NewServeMux()
	mux.HandleFunc("/", ServerHandler)
	server := &Server{
		Addr:    "localhost:4040",
		Handler: mux,
	}
	go server.ListenAndServeTLSSpdyOnly(SERVER_CERTFILE, SERVER_KEYFILE)
	time.Sleep(400 * time.Millisecond)

	//client
	cert, err := tls.LoadX509KeyPair(CLIENT_CERTFILE, CLIENT_KEYFILE)
	if err != nil {
		t.Fatal(err.Error())
	}
	config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true}
	conn, err := tls.Dial("tcp", "127.0.0.1:4040", &config)
	if err != nil {
		t.Fatal(err.Error())
	}
	client, err := NewClientConn(conn)
	if err != nil {
		t.Fatal(err.Error())
	}
	req, err := http.NewRequest("GET", "http://localhost:4040/banana", nil)
	if err != nil {
		t.Fatal(err.Error())
	}
	res, err := client.Do(req)
	if err != nil {
		t.Fatal(err.Error())
	}
	data := make([]byte, int(res.ContentLength))
	_, err = res.Body.(io.Reader).Read(data)
	if string(data) != "Hi there, I love banana!" {
		t.Fatal("Unexpected Data")
	}
	res.Body.Close()

	//close client
	err = client.Close()
	if err != nil {
		t.Fatal(err.Error())
	}
	//server close
	server.Close()
	time.Sleep(100 * time.Millisecond)
}
