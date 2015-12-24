package http

import (
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/valyala/fasthttp"
	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
)

// === Gun ===

type FastHttpGun struct {
	target string
	ssl    bool
	client *fasthttp.Client
}

// Shoot to target, this method is not thread safe
func (hg *FastHttpGun) Shoot(ctx context.Context, a ammo.Ammo,
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
	var res fasthttp.Response
	switch ha.Method {
	case "GET":
		var req fasthttp.Request
		req.SetRequestURI(uri)
		err := hg.client.Do(&req, &res)
		if err != nil {
			log.Printf("Error performing a request: %s\n", err)
			ss.err = err
			return err
		}
	default:
		log.Printf("Method not implemented: %s\n", ha.Method)
	}

	// TODO: make this an optional verbose answ_log output
	//data := make([]byte, int(res.ContentLength))
	// _, err = res.Body.(io.Reader).Read(data)
	// fmt.Println(string(data))
	ss.StatusCode = res.StatusCode()
	return nil
}

func (hg *FastHttpGun) Close() {
	hg.client = nil
}

func (hg *FastHttpGun) Connect(results chan<- aggregate.Sample) {
	hg.Close()
	// config := tls.Config{
	// 	InsecureSkipVerify: true,
	// }
	// TODO: do we want to give access to keep alive settings for guns in config?
	// dialer := &net.Dialer{
	// 	Timeout:   dialTimeout * time.Second,
	// 	KeepAlive: 120 * time.Second,
	// }
	// tr := &fasthttp.Transport{
	// 	TLSClientConfig:     &config,
	// 	Dial:                dialer.Dial,
	// 	TLSHandshakeTimeout: dialTimeout * time.Second,
	// }
	hg.client = &fasthttp.Client{}
}
