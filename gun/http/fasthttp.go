package http

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/net/context"

	"github.com/valyala/fasthttp"
	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/utils"
)

// === Gun ===

type FastHttpGun struct {
	target  string
	ssl     bool
	results chan<- *aggregate.Sample
	client  *fasthttp.HostClient
}

// Shoot to target, this method is not thread safe
func (hg *FastHttpGun) Shoot(ctx context.Context, a ammo.Ammo) error {

	if hg.client == nil {
		hg.Connect()
	}

	start := time.Now()
	ss := aggregate.AcquireSample(float64(start.UnixNano())/1e9, "REQUEST")

	ha, ok := a.(*ammo.Http)
	if !ok {
		errStr := fmt.Sprintf("Got '%T' instead of 'HttpAmmo'", a)
		log.Println(errStr)
		err := errors.New(errStr)
		ss.Err = err
		return err
	}
	if ha.Tag != "" {
		ss.Tag += "|" + ha.Tag
	}

	res := fasthttp.AcquireResponse()
	switch ha.Method {
	case "GET":
		req := fasthttp.AcquireRequest()
		req.SetRequestURI(ha.Uri)
		for k, v := range ha.Headers {
			req.Header.Set(k, v)
		}
		err := hg.client.Do(req, res)
		fasthttp.ReleaseRequest(req)
		if err != nil {
			log.Printf("Error performing a request: %s\n", err)
			fasthttp.ReleaseResponse(res)
			ss.Err = err
			ss.RT = int(time.Since(start).Seconds() * 1e6)
			hg.results <- ss
			return err
		}
	default:
		log.Printf("Method not implemented: %s\n", ha.Method)
	}

	// TODO: optional verbose answ_log output

	ss.ProtoCode = res.StatusCode()
	ss.RT = int(time.Since(start).Seconds() * 1e6)
	hg.results <- ss
	fasthttp.ReleaseResponse(res)
	return nil
}

func (hg *FastHttpGun) BindResultsTo(results chan<- *aggregate.Sample) {
	hg.results = results
}

func (hg *FastHttpGun) Close() {
	hg.client = nil
}

func (hg *FastHttpGun) Connect() {
	hg.Close()
	config := tls.Config{
		InsecureSkipVerify: true,
	}

	hg.client = &fasthttp.HostClient{
		Addr:      hg.target,
		Name:      "Pandora/" + utils.Version,
		IsTLS:     hg.ssl,
		TLSConfig: &config,
		MaxConns:  100000,
	}
}
