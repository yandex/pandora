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

func (sg *SpdyGun) Shoot(ctx context.Context, a ammo.Ammo, results chan<- *aggregate.Sample) error {
	if sg.client == nil {
		if err := sg.Connect(results); err != nil {
			return err
		}
	}

	start := time.Now()
	ss := aggregate.AcquireSample(float64(start.UnixNano())/1e9, "REQUEST")
	defer func() {
		ss.RT = int(time.Since(start).Seconds() * 1e6)
		results <- ss
	}()
	// now send the request to obtain a http response
	ha, ok := a.(*ammo.Http)
	if !ok {
		panic(fmt.Sprintf("Got '%T' instead of 'HttpAmmo'", a))
	}
	if ha.Tag != "" {
		ss.Tag += "|" + ha.Tag
	}

	req, err := http.NewRequest(ha.Method, "https://"+ha.Host+ha.Uri, nil)
	if err != nil {
		log.Printf("Error making HTTP request: %s\n", err)
		ss.Err = err
		ss.NetCode = 999
		return err
	}
	res, err := sg.client.Do(req)
	if err != nil {
		log.Printf("Error performing a request: %s\n", err)
		ss.Err = err
		ss.NetCode = 999
		return err
	}

	defer res.Body.Close()
	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		log.Printf("Error reading response body: %s\n", err)
		ss.Err = err
		ss.NetCode = 999
		return err
	}
	// TODO: make this an optional verbose answ_log output
	//data := make([]byte, int(res.ContentLength))
	// _, err = res.Body.(io.Reader).Read(data)
	// fmt.Println(string(data))
	ss.ProtoCode = res.StatusCode
	return err
}

func (sg *SpdyGun) Close() {
	if sg.client != nil {
		sg.client.Close()
	}
}

func (sg *SpdyGun) Connect(results chan<- *aggregate.Sample) error {
	// FIXME: rewrite connection logic, it isn't thread safe right now.
	start := time.Now()
	ss := aggregate.AcquireSample(float64(start.UnixNano())/1e9, "CONNECT")
	defer func() {
		ss.RT = int(time.Since(start).Seconds() * 1e6)
		results <- ss
	}()
	config := tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"spdy/3.1"},
	}
	conn, err := tls.Dial("tcp", sg.target, &config)
	if err != nil {
		ss.Err = err
		ss.NetCode = 999
		return err
	}
	client, err := spdy.NewClientConn(conn)
	if err != nil {
		ss.Err = err
		ss.NetCode = 999
		return err
	} else {
		ss.ProtoCode = 200
	}
	if sg.client != nil {
		sg.Close()
	}
	sg.client = client

	return nil
}

func (sg *SpdyGun) Ping(results chan<- *aggregate.Sample) {
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
	ss := aggregate.AcquireSample(float64(pingStart.UnixNano())/1e9, "PING")
	ss.RT = int(time.Since(pingStart).Seconds() * 1e6)

	if err == nil && pinged {
		ss.ProtoCode = 200
	} else {
		ss.Err = err
		ss.ProtoCode = 500
	}
	results <- ss
	if err != nil {
		sg.Connect(results)
	}
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
	// TODO: implement this logic somewhere
	// if pingPeriod > 0 {
	// 	go func() {
	// 		for range time.NewTicker(pingPeriod).C {
	// 			if g.closed {
	// 				return
	// 			}
	// 			g.Ping(results)
	// 		}
	// 	}()
	// }
	return g, nil
}
