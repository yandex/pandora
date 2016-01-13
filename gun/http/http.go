package http

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"golang.org/x/net/context"

	"net"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/gun"
)

// === Gun ===

const (
	// TODO: extract to config?
	dialTimeout = 3 // in sec
)

type HttpGun struct {
	target string
	ssl    bool
	client *http.Client
}

// Shoot to target, this method is not thread safe
func (hg *HttpGun) Shoot(ctx context.Context, a ammo.Ammo,
	results chan<- aggregate.Sample) error {

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
	ha, ok := a.(*ammo.Http)
	if !ok {
		errStr := fmt.Sprintf("Got '%T' instead of 'HttpAmmo'", a)
		log.Println(errStr)
		ss.err = errors.New(errStr)
		return ss.err
	}
	if ha.Tag != "" {
		ss.tag += "|" + ha.Tag
	}
	var uri string
	if hg.ssl {
		uri = "https://" + ha.Host + ha.Uri
	} else {
		uri = "http://" + ha.Host + ha.Uri
	}
	req, err := http.NewRequest(ha.Method, uri, nil)
	if err != nil {
		log.Printf("Error making HTTP request: %s\n", err)
		ss.err = err
		return err
	}
	for k, v := range ha.Headers {
		req.Header.Set(k, v)
	}
	req.URL.Host = hg.target
	res, err := hg.client.Do(req)
	if err != nil {
		log.Printf("Error performing a request: %s\n", err)
		ss.err = err
		return err
	}
	defer res.Body.Close()
	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		log.Printf("Error reading response body: %s\n", err)
		ss.err = err
		return err
	}

	// TODO: make this an optional verbose answ_log output
	//data := make([]byte, int(res.ContentLength))
	// _, err = res.Body.(io.Reader).Read(data)
	// fmt.Println(string(data))
	ss.StatusCode = res.StatusCode
	return nil
}

func (hg *HttpGun) Close() {
	hg.client = nil
}

func (hg *HttpGun) Connect(results chan<- aggregate.Sample) {
	hg.Close()
	config := tls.Config{
		InsecureSkipVerify: true,
	}
	// TODO: do we want to give access to keep alive settings for guns in config?
	dialer := &net.Dialer{
		Timeout:   dialTimeout * time.Second,
		KeepAlive: 120 * time.Second,
	}
	tr := &http.Transport{
		TLSClientConfig:     &config,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: dialTimeout * time.Second,
	}
	hg.client = &http.Client{Transport: tr}
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

func (ds *HttpSample) PhoutSample() *aggregate.PhoutSample {
	var protoCode, netCode int
	if ds.err != nil {
		protoCode = 500
		netCode = 999
		log.Printf("Error code. %v\n", ds.err)
	} else {
		netCode = 0
		protoCode = ds.StatusCode
	}
	return &aggregate.PhoutSample{
		TS:            ds.ts,
		Tag:           ds.tag,
		RT:            ds.rt,
		Connect:       0,
		Send:          0,
		Latency:       0,
		Receive:       0,
		IntervalEvent: 0,
		Egress:        0,
		Igress:        0,
		NetCode:       netCode,
		ProtoCode:     protoCode,
	}
}

func (ds *HttpSample) String() string {
	return fmt.Sprintf("rt: %d [%d] %s", ds.rt, ds.StatusCode, ds.tag)
}

func New(c *config.Gun) (gun.Gun, error) {
	params := c.Parameters
	if params == nil {
		return nil, errors.New("Parameters not specified")
	}
	target, ok := params["Target"]
	if !ok {
		return nil, errors.New("Target not specified")
	}
	g := &HttpGun{}
	switch t := target.(type) {
	case string:
		g.target = target.(string)
	default:
		return nil, fmt.Errorf("Target is of the wrong type."+
			" Expected 'string' got '%T'", t)
	}
	if ssl, ok := params["SSL"]; ok {
		if sslVal, casted := ssl.(bool); casted {
			g.ssl = sslVal
		} else {
			return nil, fmt.Errorf("SSL should be boolean type.")
		}
	}
	return g, nil
}
