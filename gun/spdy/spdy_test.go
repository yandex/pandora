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
	"github.com/yandex/pandora/utils"
	"golang.org/x/net/context"
)

func TestSpdyGun(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	result := make(chan interface{})

	gun := &SpdyGun{
		target:     "localhost:3000",
		pingPeriod: time.Second * 5,
	}
	promise := utils.Promise(func() error {
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
		}, result)
	})

	results := aggregate.Drain(ctx, result)
	require.Len(t, results, 2)
	{
		// first result is connect

		sample, casted := (results[0]).(*aggregate.Sample)
		require.True(t, casted, "Should be *aggregate.Sample")
		assert.Equal(t, "CONNECT", sample.Tag)
		assert.Equal(t, 200, sample.ProtoCode)
	}
	{
		// second result is request

		sample, casted := (results[1]).(*aggregate.Sample)
		require.True(t, casted, "Should be *aggregate.Sample")
		assert.Equal(t, "REQUEST", sample.Tag)
		assert.Equal(t, 200, sample.ProtoCode)
	}

	// TODO: test scenaries with errors
	// TODO: test ping logic

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

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
