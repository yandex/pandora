// Copyright 2013-14, Amahi. All rights reserved.
// Use of this source code is governed by the
// license that can be found in the LICENSE file.

// lower level test functions

package spdy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"testing"
	"time"
)

func init() {
	SetLog(ioutil.Discard)
}

func ServerTestHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func TestFrames(t *testing.T) {
	//make server
	mux := http.NewServeMux()
	mux.HandleFunc("/", ServerTestHandler)
	server := &Server{
		Addr:    "localhost:4040",
		Handler: mux,
	}
	go server.ListenAndServe()
	time.Sleep(200 * time.Millisecond)

	//make client
	client, err := NewClient("localhost:4040")
	if err != nil {
		t.Fatal(err.Error())
	}

	//now send requests and test
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

	//another request
	req, err = http.NewRequest("POST", "http://localhost:4040/monkeys", bytes.NewBufferString("hello=world"))
	if err != nil {
		t.Fatal(err.Error())
	}

	res, err = client.Do(req)
	if err != nil {
		t.Fatal(err.Error())
	}
	data = make([]byte, int(res.ContentLength))
	_, err = res.Body.(io.Reader).Read(data)
	if string(data) != "Hi there, I love monkeys!" {
		t.Fatal("Unexpected Data")
	}
	res.Body.Close()

	//settings frame test
	set := new(settings)
	var svp []settingsValuePairs
	svp = append(svp, settingsValuePairs{flags: 0, id: 4, value: 6})   //set SETTINGS_MAX_CONCURRENT_STREAMS to 6
	svp = append(svp, settingsValuePairs{flags: 0, id: 3, value: 400}) //set SETTINGS_ROUND_TRIP_TIME to 400ms
	set.flags = 0
	set.count = 2
	set.svp = svp
	buf := new(bytes.Buffer)
	err = binary.Write(buf, binary.BigEndian, &set.count)
	if err != nil {
		t.Fatal(err.Error())
	}

	for i := uint32(0); i < set.count; i++ {
		err = binary.Write(buf, binary.BigEndian, &set.svp[i].flags)
		if err != nil {
			t.Fatal(err.Error())
		}
		err = binary.Write(buf, binary.BigEndian, &set.svp[i].id)
		if err != nil {
			t.Fatal(err.Error())
		}
		err = binary.Write(buf, binary.BigEndian, &set.svp[i].value)
		if err != nil {
			t.Fatal(err.Error())
		}
	}
	settings_frame := controlFrame{kind: FRAME_SETTINGS, flags: 0, data: buf.Bytes()}
	client.ss.out <- settings_frame
	time.Sleep(200 * time.Millisecond)

	//rstStreamtest - first, start stream on client
	//FIXME - need to add a proper test
	str := client.ss.NewClientStream()
	if str == nil {
		t.Fatal("ERROR in NewClientStream: cannot create stream")
		return
	}
	str.sendRstStream()

	//ping test
	ping, err := client.Ping(time.Second)
	if err != nil {
		t.Fatal(err.Error())
	}

	if ping == false {
		t.Fatal("Unable to ping server from client")
	}

	//close client
	err = client.Close()
	if err != nil {
		t.Fatal(err.Error())
	}
	//server close
	server.Close()
}

func TestGoaway(t *testing.T) {
	//make server
	mux := http.NewServeMux()
	mux.HandleFunc("/", ServerTestHandler)
	server_session_chan := make(chan *Session)
	server := &Server{
		Addr:    "localhost:4040",
		Handler: mux,
		ss_chan: server_session_chan,
	}
	go server.ListenAndServe()
	time.Sleep(200 * time.Millisecond)

	//make client
	client, err := NewClient("localhost:4040")
	if err != nil {
		t.Fatal(err.Error())
	}

	//prepare a request
	request, err := http.NewRequest("POST", "http://localhost:4040/banana", bytes.NewBufferString("hello world"))
	if err != nil {
		t.Fatal(err.Error())
	}

	ss := <-server_session_chan

	client_stream1 := client.ss.NewClientStream()
	err = client_stream1.prepareRequestHeader(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	f := frameSynStream{session: client_stream1.session, stream: client_stream1.id, header: request.Header, flags: 2}
	client_stream1.session.out <- f

	client_stream2 := client.ss.NewClientStream()
	err = client_stream2.prepareRequestHeader(request)
	if err != nil {
		t.Fatal(err.Error())
	}
	f = frameSynStream{session: client_stream2.session, stream: client_stream2.id, header: request.Header, flags: 2}
	client_stream2.session.out <- f
	time.Sleep(200 * time.Millisecond)

	dat := []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00}
	ss.out <- controlFrame{kind: FRAME_GOAWAY, flags: 0, data: dat}
	time.Sleep(200 * time.Millisecond)

	if client.ss.NewClientStream() != nil {
		t.Fatal("Stream Made even after goaway sent")
	}

	if client_stream1.closed == true {
		t.Fatal("Stream#1 closed: unexpected")
	}

	if client_stream2.closed == false {
		t.Fatal("Stream#2 alive: unexpected")
	}

	//close client
	err = client.Close()
	if err != nil {
		t.Fatal(err.Error())
	}
	//server close
	server.Close()
}
