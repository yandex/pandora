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
	pingPeriod time.Duration
	target     string
	client     *spdy.Client
}

func (sg *SpdyGun) Run(a Ammo, results chan<- Sample) {
	if sg.client == nil {
		sg.Connect(results)
	}
	if sg.pingPeriod > 0 {
		pingTimer := time.NewTicker(sg.pingPeriod)
		go func() {
			for range pingTimer.C {
				sg.Ping(results)
			}
		}()
	}
	start := time.Now()
	ss := &SpdySample{ts: float64(start.UnixNano()) / 1e9, tag: "REQUEST"}
	defer func() {
		ss.rt = int(time.Since(start).Seconds() * 1e6)
		results <- ss
	}()
	// now send the request to obtain a http response
	ha, ok := a.(*HttpAmmo)
	if !ok {
		errStr := fmt.Sprintf("Got '%T' instead of 'HttpAmmo'", a)
		log.Println(errStr)
		ss.err = errors.New(errStr)
		return
	}
	if ha.Tag != "" {
		ss.tag += "|" + ha.Tag
	}
	req, err := ha.Request()
	if err != nil {
		log.Printf("Error making HTTP request: %s\n", err)
		ss.err = err
		return
	}
	res, err := sg.client.Do(req)
	if err != nil {
		log.Printf("Error performing a request: %s\n", err)
		ss.err = err
		return
	}
	defer res.Body.Close()
	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		log.Printf("Error reading response body: %s\n", err)
		ss.err = err
		return
	}

	// TODO: make this an optional verbose answ_log output
	//data := make([]byte, int(res.ContentLength))
	// _, err = res.Body.(io.Reader).Read(data)
	// fmt.Println(string(data))
	ss.StatusCode = res.StatusCode
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
		log.Printf("client: dial: %s\n", err)
		return
	}
	sg.client, err = spdy.NewClientConn(conn)
	if err != nil {
		log.Printf("client: connect: %s\n", err)
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

func (sg *SpdyGun) Ping(results chan<- Sample) {
	if sg.client == nil { // TODO: this might be a faulty behaviour
		sg.Connect(results)
	}
	pingStart := time.Now()

	pinged, err := sg.client.Ping(time.Second * 15)
	if err != nil {
		log.Printf("client: ping: %s\n", err)
	}
	if !pinged {
		log.Printf("client: ping: timed out\n")
	}
	ss := &SpdySample{ts: float64(pingStart.UnixNano()) / 1e9, tag: "PING"}
	ss.rt = int(time.Since(pingStart).Seconds() * 1e6)
	ss.err = err
	if ss.err == nil && pinged {
		ss.StatusCode = 200
	} else {
		ss.StatusCode = 500
	}
	results <- ss
	if err != nil {
		sg.Connect(results)
	}
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
	var pingPeriod time.Duration
	paramPingPeriod, ok := params["PingPeriod"]
	if !ok {
		paramPingPeriod = 120.0 // TODO: move this default elsewhere
	}
	switch t := paramPingPeriod.(type) {
	case float64:
		pingPeriod = time.Duration(paramPingPeriod.(float64)*1e3) * time.Millisecond
	default:
		return nil, errors.New(fmt.Sprintf("Period is of the wrong type."+
			" Expected 'float64' got '%T'", t))
	}
	switch t := target.(type) {
	case string:
		g = &SpdyGun{
			pingPeriod: pingPeriod,
			target:     target.(string),
		}
	default:
		return nil, errors.New(fmt.Sprintf("Target is of the wrong type."+
			" Expected 'string' got '%T'", t))
	}
	return g, nil
}
