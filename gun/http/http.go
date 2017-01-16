package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/gun"
)

const (
	// TODO: extract to config?
	dialTimeout = 3 // in sec
)

func New(conf Config) *HttpGun {
	return &HttpGun{config: conf}
}

type Config struct {
	Target string
	SSL    bool
}

type HttpGun struct {
	config  Config
	client  *http.Client
	results chan<- *aggregate.Sample
}

var _ gun.Gun = (*HttpGun)(nil)

func (g *HttpGun) BindResultsTo(results chan<- *aggregate.Sample) {
	g.results = results
}

// Shoot to target, this method is not thread safe
func (g *HttpGun) Shoot(ctx context.Context, a ammo.Ammo) (err error) {
	if g.client == nil {
		g.Connect()
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
	var uri string

	// TODO: get rid of ha.Host that is overwrite by gh target
	if g.config.SSL {
		uri = "https://" + ha.Host + ha.Uri
	} else {
		uri = "http://" + ha.Host + ha.Uri
	}
	var req *http.Request
	req, err = http.NewRequest(ha.Method, uri, nil)
	if err != nil {
		log.Printf("Error making HTTP request: %s\n", err)
		return
	}
	for k, v := range ha.Headers {
		req.Header.Set(k, v)
	}
	req.URL.Host = g.config.Target
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
	return
}

func (g *HttpGun) Close() {
	g.client = nil
}

func (g *HttpGun) Connect() {
	g.Close()
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
	g.client = &http.Client{Transport: tr}
}
