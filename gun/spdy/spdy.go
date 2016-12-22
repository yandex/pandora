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
	"context"

	"github.com/amahi/spdy"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/gun"
)

type SPDYGun struct {
	target  string
	client  *spdy.Client
	results chan<- *aggregate.Sample
}

func (g *SPDYGun) BindResultsTo(results chan<- *aggregate.Sample) {
	g.results = results
}

func (g *SPDYGun) Shoot(ctx context.Context, a ammo.Ammo) (err error) {
	if g.client == nil {
		if err = g.Connect(); err != nil {
			return err
		}
	}
	ss := aggregate.AcquireSample("REQUEST")
	defer func() {
		if err != nil {
			ss.SetErr(err)
		}
		g.results <- ss
	}()
	// now send the request to obtain a http response
	ha, ok := a.(*ammo.Http)
	if !ok {
		panic(fmt.Sprintf("Got '%T' instead of 'HttpAmmo'", a))
	}
	if ha.Tag != "" {
		ss.AddTag(ha.Tag)
	}

	var req *http.Request
	req, err = http.NewRequest(ha.Method, "https://"+ha.Host+ha.Uri, nil)
	if err != nil {
		log.Printf("Error making HTTP request: %s\n", err)
		return
	}

	var res *http.Response
	res, err = g.client.Do(req)
	if err != nil {
		log.Printf("Error performing a request: %s\n", err)
		return
	}

	defer res.Body.Close()
	_, err = io.Copy(ioutil.Discard, res.Body)
	if err != nil {
		log.Printf("Error reading response body: %s\n", err)
		return
	}
	// TODO: make this an optional verbose answ_log output
	//data := make([]byte, int(res.ContentLength))
	// _, err = res.Body.(io.Reader).Read(data)
	// fmt.Println(string(data))
	ss.SetProtoCode(res.StatusCode)
	return err
}

func (g *SPDYGun) Close() {
	if g.client != nil {
		g.client.Close()
		g.client = nil
	}
}

func (g *SPDYGun) Connect() (err error) {
	// FIXME: rewrite connection logic, it isn't thread safe right now.
	ss := aggregate.AcquireSample("CONNECT")
	defer func() {
		if err != nil {
			ss.SetErr(err)
		}
		g.results <- ss
	}()
	config := tls.Config{
		InsecureSkipVerify: true,
		NextProtos:         []string{"spdy/3.1"},
	}
	conn, err := tls.Dial("tcp", g.target, &config)
	if err != nil {
		return
	}
	client, err := spdy.NewClientConn(conn)
	if err != nil {
		return err
	} else {
		ss.SetProtoCode(http.StatusOK)
	}
	if g.client != nil {
		g.Close()
	}
	g.client = client

	return nil
}

func (sg *SPDYGun) Ping() {
	if sg.client == nil {
		return
	}
	ss := aggregate.AcquireSample("PING")
	pinged, err := sg.client.Ping(time.Second * 15)
	if err != nil {
		log.Printf("Client: ping: %s\n", err)
	}
	if !pinged {
		log.Println("Client: ping: timed out")
	}
	if err == nil && pinged {
		ss.SetProtoCode(http.StatusOK)
	} else {
		ss.SetErr(err)
	}
	sg.results <- ss
	if err != nil {
		sg.Connect()
	}
}

func (sg *SPDYGun) startAutoPing(pingPeriod time.Duration) {
	if pingPeriod > 0 {
		go func() {
			for range time.NewTicker(pingPeriod).C {
				if sg.client == nil {
					return
				}
				sg.Ping()
			}
		}()
	}
}

func New(c *config.Gun) (gun.Gun, error) {
	// TODO: use mapstrucuture to do such things
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
		g = &SPDYGun{
			target: target.(string),
		}
	default:
		return nil, fmt.Errorf("Target is of the wrong type."+
			" Expected 'string' got '%T'", t)
	}
	g.(*SPDYGun).startAutoPing(pingPeriod)
	return g, nil
}
