package main

import (
        "crypto/tls"
	"fmt"
	"github.com/amahi/spdy"
	"io"
	"net/http"
)

func handle(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
        cert, err := tls.LoadX509KeyPair("client.pem", "client.key")
        if err != nil {
                fmt.Printf("server: loadkeys: %s", err)
        }
        config := tls.Config{Certificates: []tls.Certificate{cert}, InsecureSkipVerify: true, NextProtos: []string{"spdy/3"}}
        conn, err := tls.Dial("tcp", "127.0.0.1:4040", &config)
        if err != nil {
                fmt.Printf("client: dial: %s", err)
        }
	client, err := spdy.NewClientConn(conn)
	handle(err)
	req, err := http.NewRequest("GET", "http://localhost:4040/banana", nil)
	handle(err)
	res, err := client.Do(req)
	handle(err)
	data := make([]byte, int(res.ContentLength))
	_, err = res.Body.(io.Reader).Read(data)
	fmt.Println(string(data))
	res.Body.Close()
}
