package spdy

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/amahi/spdy"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/gun"
	"golang.org/x/net/context"
)

type SpdyGun struct {
	pingPeriod time.Duration
	target     string
	client     *spdy.Client
}

func (sg *SpdyGun) Shoot(ctx context.Context, a ammo.Ammo, results chan<- aggregate.Sample) error {
	if sg.client == nil {
		if err := sg.connect(results); err != nil {
			return err
		}
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

	req, err := http.NewRequest(ha.Method, "https://"+ha.Host+ha.Uri, nil)
	if err != nil {
		log.Printf("Error making HTTP request: %s\n", err)
		ss.err = err
		return err
	}
	res, err := sg.client.Do(req)
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
	return err
}

func (sg *SpdyGun) Close() {
	if sg.client != nil {
		sg.client.Close()
	}
}

func (sg *SpdyGun) connect(results chan<- aggregate.Sample) error {
	// FIXME: rewrite connection logic, it isn't thread safe right now.
	connectStart := time.Now()
	config := tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"spdy/3.1"},
	}
	conn, err := tls.Dial("tcp", sg.target, &config)
	if err != nil {
		return fmt.Errorf("client: dial: %s\n", err)
	}
	client, err := spdy.NewClientConn(conn)
	if err != nil {
		return fmt.Errorf("client: connect: %v\n", err)
	}
	if sg.client != nil {
		sg.Close()
	}
	sg.client = client
	ss := &SpdySample{ts: float64(connectStart.UnixNano()) / 1e9, tag: "CONNECT"}
	ss.rt = int(time.Since(connectStart).Seconds() * 1e6)
	ss.err = err
	if ss.err == nil {
		ss.StatusCode = 200
	}
	results <- ss
	return nil
}

func (sg *SpdyGun) Ping(results chan<- aggregate.Sample) {
	if sg.client == nil {
		return
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
		sg.connect(results)
	}
}

type SpdySample struct {
	ts         float64 // Unix Timestamp in seconds
	rt         int     // response time in milliseconds
	StatusCode int     // protocol status code
	tag        string
	err        error
}

func (ds *SpdySample) PhoutSample() *aggregate.PhoutSample {
	var protoCode, netCode int
	if ds.err != nil {
		protoCode = 500
		netCode = 999
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

func (ds *SpdySample) String() string {
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
	var pingPeriod time.Duration
	paramPingPeriod, ok := params["PingPeriod"]
	if !ok {
		paramPingPeriod = 120.0 // TODO: move this default elsewhere
	}
	switch t := paramPingPeriod.(type) {
	case float64:
		pingPeriod = time.Duration(paramPingPeriod.(float64)*1e3) * time.Millisecond
	default:
		return nil, fmt.Errorf("Period is of the wrong type."+
			" Expected 'float64' got '%T'", t)
	}
	var g gun.Gun
	switch t := target.(type) {
	case string:
		g = &SpdyGun{
			pingPeriod: pingPeriod,
			target:     target.(string),
		}
	default:
		return nil, fmt.Errorf("Target is of the wrong type."+
			" Expected 'string' got '%T'", t)
	}
	return g, nil
}
