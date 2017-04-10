package phttp

import (
	"log"
	"net/http"
	"testing"

	"github.com/SlyMarbo/spdy" // we specially use SPDY server from another library
)

func runSPDYTestServer() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("This is an example server.\n"))
	})

	//use SPDY's Listen and serve
	log.Println("Run SPDY server on localhost:3000")
	err := spdy.ListenAndServeTLS("localhost:3000",

		"./testdata/test.crt", "./testdata/test.key", nil)
	if err != nil {
		//error handling here
		log.Panic(err)
	}
}

func TestSPDYGun(t *testing.T) {
	t.Skip("NIY") // TODO
	//go runSPDYTestServer()
	//// TODO: try connect in cycle, break on success.
	//time.Sleep(time.Millisecond * 5) // wait for server

	t.Run("Shoot", func(t *testing.T) {
		//ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		//defer cancel()
		//
		//gun := &SPDYGun{
		//config: SPDYGunConfig{
		//	Target: "localhost:3000",
		//},
		//results: result,
		//}

		//promise := utils.Promise(func() error {
		//	defer gun.Close()
		//	defer close(result)
		//	return gun.Shoot(ctx, nil)
		// &ammo.jsonlineAmmo{
		//	Host:   "example.org",
		//	Method: "GET",
		//	Uri:    "/path",
		//	Headers: map[string]string{
		//		"Accept":          "*/*",
		//		"Accept-Encoding": "gzip, deflate",
		//		"Host":            "example.org",
		//		"User-Agent":      "Pandora/0.0.1",
		//	},
		//})
		//})

		//results := aggregate.Drain(ctx, result)
		//require.Len(t, results, 2)
		//// first result is connect
		//assert.Equal(t, "CONNECT", results[0].Tags())
		//assert.Equal(t, 200, results[0].ProtoCode())
		//// second result is request
		//assert.Equal(t, "REQUEST", results[1].Tags())
		//assert.Equal(t, 200, results[1].ProtoCode())

		// TODO: test scenaries with errors

		//select {
		//case err := <-promise:
		//	require.NoError(t, err)
		//case <-ctx.Done():
		//	t.Fatal(ctx.Err())
		//}
	})

	t.Run("ConnectPing", func(t *testing.T) {

		//ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		//defer cancel()

		//result := make(chan *aggregate.Sample)
		//
		//gun := &SPDYGun{
		////config: SPDYGunConfig{
		////	Target: "localhost:3000",
		////},
		////results: result,
		//}
		//promise := utils.Promise(func() error {
		//	defer gun.Close()
		//	defer close(result)
		//	if err := gun.Connect(context.Background()); err != nil {
		//		return err
		//	}
		//	gun.Ping()
		//	return nil
		//})
		//
		//results := aggregate.Drain(ctx, result)
		//require.Len(t, results, 2)
		//// first result is connect
		//assert.Equal(t, "CONNECT", results[0].Tags())
		//assert.Equal(t, 200, results[0].ProtoCode())
		//// second result is PING
		//assert.Equal(t, "PING", results[1].Tags())
		//assert.Equal(t, 200, results[1].ProtoCode())
		//select {
		//case err := <-promise:
		//	require.NoError(t, err)
		//case <-ctx.Done():
		//	t.Fatal(ctx.Err())
		//}
	})

}
