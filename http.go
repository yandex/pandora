package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type HttpAmmo struct {
	Host    string
	Method  string
	Uri     string
	Headers map[string]string
	Tag     string
}

type HttpAmmoJsonDecoder struct{}

func (ha *HttpAmmoJsonDecoder) FromString(jsonDoc string) (a Ammo, err error) {
	a = &HttpAmmo{}
	err = json.Unmarshal([]byte(jsonDoc), a)
	return
}

func (ha *HttpAmmo) Request() (req *http.Request, err error) {
	//make a request
	req, err = http.NewRequest(ha.Method, "https://"+ha.Host+ha.Uri, nil)
	for k, v := range ha.Headers {
		req.Header.Set(k, v)
	}
	return
}

type HttpAmmoProvider struct {
	ammoProvider
	ammoFile  *os.File
	ammoLimit int
	loopLimit int
}

func (ap *HttpAmmoProvider) Start() {
	go func() { // requests reader/generator
		ammoNumber := 0
		loops := 0
		for {
			scanner := bufio.NewScanner(ap.ammoFile)
			scanner.Split(bufio.ScanLines)
			for scanner.Scan() && (ap.ammoLimit == 0 || ammoNumber < ap.ammoLimit) {
				txt := scanner.Text()
				if a, err := ap.decoder.FromString(txt); err != nil {
					log.Fatal("Failed to decode ammo: ", err)
				} else {
					ammoNumber += 1
					ap.source <- a
				}
			}
			if loops > ap.loopLimit {
				break
			}
			ap.ammoFile.Seek(0, 0)
			log.Printf("Restarted ammo the beginning. Loops left: %d\n", ap.loopLimit-loops)
			loops++
		}
		close(ap.source)
		log.Println("Ran out of ammo")
	}()
}

func NewHttpAmmoProvider(filename string, ammoLimit int, loopLimit int) (ap AmmoProvider, err error) {
	file, err := os.Open(filename)
	if err == nil {
		ap = &HttpAmmoProvider{
			ammoLimit: ammoLimit,
			loopLimit: loopLimit,
			ammoFile:  file,
			ammoProvider: ammoProvider{
				decoder: &HttpAmmoJsonDecoder{},
				source:  make(chan Ammo, 128),
			},
		}
	}
	return
}

// === Gun ===

type HttpGun struct {
	target string
	client *http.Client
}

func (hg *HttpGun) Run(a Ammo, results chan<- Sample) {
	if hg.client == nil {
		hg.Connect(results)
	}
	start := time.Now()
	ss := &HttpSample{ts: float64(start.UnixNano()) / 1e9, tag: "REQUEST"}
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
	res, err := hg.client.Do(req)
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

func (hg *HttpGun) Close() {
	hg.client = nil
}

func (hg *HttpGun) Connect(results chan<- Sample) {
	hg.Close()
	hg.client = &http.Client{}
	// 	connectStart := time.Now()
	// 	config := tls.Config{
	// 		InsecureSkipVerify: true,
	// 		NextProtos:         []string{"HTTP/1.1"},
	// 	}

	// 	conn, err := tls.Dial("tcp", hg.target, &config)
	// 	if err != nil {
	// 		log.Printf("client: dial: %s\n", err)
	// 		return
	// 	}
	// 	hg.client, err = Http.NewClientConn(conn)
	// 	if err != nil {
	// 		log.Printf("client: connect: %s\n", err)
	// 		return
	// 	}
	// 	ss := &HttpSample{ts: float64(connectStart.UnixNano()) / 1e9, tag: "CONNECT"}
	// 	ss.rt = int(time.Since(connectStart).Seconds() * 1e6)
	// 	ss.err = err
	// 	if ss.err == nil {
	// 		ss.StatusCode = 200
	// 	}
	// 	results <- ss
}

type HttpSample struct {
	ts         float64 // Unix Timestamp in seconds
	rt         int     // response time in milliseconds
	StatusCode int     // protocol status code
	tag        string
	err        error
}

func (ds *HttpSample) PhoutSample() *PhoutSample {
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

func (ds *HttpSample) String() string {
	return fmt.Sprintf("rt: %d [%d] %s", ds.rt, ds.StatusCode, ds.tag)
}

func NewHttpGunFromConfig(c *GunConfig) (g Gun, err error) {
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
		g = &HttpGun{
			target: target.(string),
		}
	default:
		return nil, errors.New(fmt.Sprintf("Target is of the wrong type."+
			" Expected 'string' got '%T'", t))
	}
	return g, nil
}
