package spdy

import (
	"flag"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/SlyMarbo/spdy" // we specially use spdy server from another library
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/utils"
	"golang.org/x/net/context"
)

func TestSpdyGun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	result := make(chan *aggregate.Sample)

	gun := &SpdyGun{
		target:  "localhost:3000",
		results: result,
	}
	promise := utils.Promise(func() error {
		defer gun.Close()
		defer close(result)
		return gun.Shoot(ctx, &ammo.Http{
			Host:   "example.org",
			Method: "GET",
			Uri:    "/path",
			Headers: map[string]string{
				"Accept":          "*/*",
				"Accept-Encoding": "gzip, deflate",
				"Host":            "example.org",
				"User-Agent":      "Pandora/0.0.1",
			},
		})
	})

	results := aggregate.Drain(ctx, result)
	require.Len(t, results, 2)
	{
		// first result is connect

		assert.Equal(t, "CONNECT", results[0].Tag)
		assert.Equal(t, 200, results[0].ProtoCode)
	}
	{
		// second result is request

		assert.Equal(t, "REQUEST", results[1].Tag)
		assert.Equal(t, 200, results[1].ProtoCode)
	}

	// TODO: test scenaries with errors

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

}

func TestSpdyConnectPing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	result := make(chan *aggregate.Sample)

	gun := &SpdyGun{
		target:  "localhost:3000",
		results: result,
	}
	promise := utils.Promise(func() error {
		defer gun.Close()
		defer close(result)
		if err := gun.Connect(); err != nil {
			return err
		}
		gun.Ping()
		return nil
	})

	results := aggregate.Drain(ctx, result)
	require.Len(t, results, 2)
	{
		// first result is connect

		assert.Equal(t, "CONNECT", results[0].Tag)
		assert.Equal(t, 200, results[0].ProtoCode)
	}
	{
		// second result is PING

		assert.Equal(t, "PING", results[1].Tag)
		assert.Equal(t, 200, results[1].ProtoCode)
	}
	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

}

func TestNewSpdyGun(t *testing.T) {
	spdyConfig := &config.Gun{
		GunType: "spdy",
		Parameters: map[string]interface{}{
			"Target":     "localhost:3000",
			"PingPeriod": 5.0,
		},
	}
	g, err := New(spdyConfig)
	assert.NoError(t, err)
	_, ok := g.(*SpdyGun)
	assert.Equal(t, true, ok)

	failSpdyConfig := &config.Gun{
		GunType: "spdy",
		Parameters: map[string]interface{}{
			"Target":     "localhost:3000",
			"PingPeriod": "not-a-number",
		},
	}
	_, err = New(failSpdyConfig)
	assert.Error(t, err)
}

func runSpdyTestServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("This is an example server.\n"))
	})

	//use spdy's Listen and serve
	log.Println("Run spdy server on localhost:3000")
	err := spdy.ListenAndServeTLS("localhost:3000",
		"./testdata/test.crt", "./testdata/test.key", nil)
	if err != nil {
		//error handling here
		log.Panic(err)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	go runSpdyTestServer()
	time.Sleep(time.Millisecond * 5) // wait for server
	os.Exit(m.Run())
}
