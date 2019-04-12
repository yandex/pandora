package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/spf13/afero"
	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/components/phttp/import"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/import"
	"github.com/yandex/pandora/core/register"
	"io/ioutil"
	"log"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Ammo struct {
	Tag string
}

type Sample struct {
	URL              string
	ShootTimeSeconds float64
}

type GunConfig struct {
	Target string `validate:"required"`
	Handler string `validate:"required"`// Configuration will fail, without target defined
}

type Gun struct {
	// Configured on construction.
	client websocket.Conn
	conf   GunConfig
	// Configured on Bind, before shooting
	aggr core.Aggregator // May be your custom Aggregator.
	core.GunDeps
}

func NewGun(conf GunConfig) *Gun {
	return &Gun{conf: conf}
}

func (g *Gun) Bind(aggr core.Aggregator, deps core.GunDeps) error {
	targetPath := url.URL{Scheme: "ws", Host: g.conf.Target, Path: g.conf.Handler}
	sample := netsample.Acquire("connection")
	code := 0
	rand.Seed(time.Now().Unix())
	conn, _, err := websocket.DefaultDialer.Dial(
		targetPath.String(),
		nil,
	)
	if err != nil {
		log.Fatalf("dial err FATAL %s:", err)
		code = 500
	} else {
		code = 200
	}
	g.client = *conn
	g.aggr = aggr
	g.GunDeps = deps
	defer func() {
		sample.SetProtoCode(code)
		g.aggr.Report(sample)
	}()

	go func() {
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				code = 400
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	err = conn.WriteMessage(websocket.TextMessage, []byte("some websocket connection initialization text, e.g. token"))
	if err != nil {
		log.Println("write:", err)
	}
	return nil
}

func (g *Gun) Shoot(ammo core.Ammo) {
	sample := netsample.Acquire("message")
	code := 0
	conn := g.client
	err := conn.WriteMessage(websocket.TextMessage, []byte("test_message"))
	if err != nil {
		log.Println("connection closed", err)
		code = 600
	} else {
		code = 200
	}
	defer func() {
		sample.SetProtoCode(code)
		g.aggr.Report(sample)
	}()

}

func main() {
	//debug.SetGCPercent(-1)
	// Standard imports.
	fs := afero.NewOsFs()
	coreimport.Import(fs)
	// May not be imported, if you don't need http guns and etc.
	phttp.Import(fs)

	// Custom imports. Integrate your custom types into configuration system.
	coreimport.RegisterCustomJSONProvider("ammo_provider", func() core.Ammo { return &Ammo{} })

	register.Gun("websocketGun", NewGun, func() GunConfig {
		return GunConfig{
			Target: "default target",
		}
	})

	cli.Run()
}