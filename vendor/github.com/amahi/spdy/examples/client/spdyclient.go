package main

import (
	"fmt"
	"github.com/amahi/spdy"
	"io"
	"net/http"
)

func main() {
        //make a spdy client with a given address
	client, err := spdy.NewClient("localhost:4040")
	if err != nil {
		//handle error here
	}
	
	//make a request
	req, err := http.NewRequest("GET", "http://localhost:4040/banana", nil)
	if err != nil {
		//handle error here
	}
	
	//now send the request to obtain a http response
	res, err := client.Do(req)
	if err != nil {
		//something went wrong
	}
	
	//now handle the response
	data := make([]byte, int(res.ContentLength))
	_, err = res.Body.(io.Reader).Read(data)
	fmt.Println(string(data))
	fmt.Println(res.Header)
	res.Body.Close()
}
