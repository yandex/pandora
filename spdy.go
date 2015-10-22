package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/amahi/spdy"
	"io"
	"io/ioutil"
	"log"
	"time"
)

type SpdyGun struct {
	target string
	client *spdy.Client
}

func (sg *SpdyGun) Run(a Ammo, results chan<- Sample) {
	if sg.client == nil {
		sg.Connect(results)
	}
	start := time.Now()
	ss := &SpdySample{ts: float64(start.UnixNano()) / 1e9, tag: "REQUEST"}
	defer func() {
		ss.rt = int(time.Since(start).Seconds() * 1e6)
		results <- ss
	}()
	// now send the request to obtain a http response
	if req, err := a.(*HttpAmmo).Request(); err != nil {
		log.Printf("Could not convert ammo to HTTP request: %s\n", err)
		ss.err = err
	} else if res, err := sg.client.Do(req); err != nil {
		log.Printf("Error performing a request: %s\n", err)
		ss.err = err
	} else if _, err := io.Copy(ioutil.Discard, res.Body); err != nil {
		log.Printf("Error reading response body: %s\n", err)
		ss.err = err
	} else {

		// TODO: make this an optional verbose answ_log output
		//data := make([]byte, int(res.ContentLength))
		// _, err = res.Body.(io.Reader).Read(data)
		// fmt.Println(string(data))
		res.Body.Close()
		ss.StatusCode = res.StatusCode
	}
	return
}

func (sg *SpdyGun) Close() {
	if sg.client != nil {
		sg.client.Close()
		sg.client = nil
	}
}

func (sg *SpdyGun) Connect(results chan<- Sample) {
	sg.Close()
	connectStart := time.Now()
	config := tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"spdy/3.1"},
	}

	conn, err := tls.Dial("tcp", sg.target, &config)
	if err != nil {
		fmt.Printf("client: dial: %s\n", err)
		return
	}
	sg.client, err = spdy.NewClientConn(conn)
	if err != nil {
		fmt.Printf("client: connect: %s\n", err)
		return
	}
	ss := &SpdySample{ts: float64(connectStart.UnixNano()) / 1e9, tag: "CONNECT"}
	ss.rt = int(time.Since(connectStart).Seconds() * 1e6)
	ss.err = err
	if ss.err == nil {
		ss.StatusCode = 200
	}
	results <- ss
}

type SpdySample struct {
	ts         float64 // Unix Timestamp in seconds
	rt         int     // response time in milliseconds
	StatusCode int     // protocol status code
	tag        string
	err        error
}

func (ds *SpdySample) PhoutSample() *PhoutSample {
	var protoCode, netCode int
	if ds.err != nil {
		protoCode = 500
		netCode = 999
	} else {
		netCode = 0
		protoCode = ds.StatusCode
	}
	return &PhoutSample{
		ts:             ds.ts,
		tag:            ds.tag,
		rt:             ds.rt,
		connect:        0,
		send:           0,
		latency:        0,
		receive:        0,
		interval_event: 0,
		egress:         0,
		igress:         0,
		netCode:        netCode,
		protoCode:      protoCode,
	}
}

func (ds *SpdySample) String() string {
	return fmt.Sprintf("rt: %d [%d] %s", ds.rt, ds.StatusCode, ds.tag)
}

func NewSpdyGunFromConfig(c *GunConfig) (g Gun, err error) {
	params := c.Parameters
	if params == nil {
		return nil, errors.New("Parameters not specified")
	}
	target, ok := params["Target"]
	if !ok {
		return nil, errors.New("Target not specified")
	}
	switch t := target.(type) {
	case string:
		g = &SpdyGun{target: target.(string)}
	default:
		return nil, errors.New(fmt.Sprintf("Target is of the wrong type."+
			" Expected 'string' got '%T'", t))
	}
	return g, nil
}
