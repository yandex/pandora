package ammo

import (
	"bufio"
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/utils"
)

const (
	httpTestFilename = "./testdata/ammo.jsonline"
)

func TestNewHttpProvider(t *testing.T) {
	c := &config.AmmoProvider{
		AmmoSource: httpTestFilename,
		AmmoLimit:  10,
	}
	provider, err := NewHttpProvider(c)
	require.NoError(t, err)

	httpProvider, casted := provider.(*HttpProvider)
	require.True(t, casted, "NewHttpProvider should return *HttpProvider type")

	// look at defaults
	assert.Equal(t, 10, httpProvider.ammoLimit)
	assert.Equal(t, 0, httpProvider.passes)
	assert.NotNil(t, httpProvider.sink)
	assert.NotNil(t, httpProvider.BaseProvider.source)
	assert.NotNil(t, httpProvider.BaseProvider.decoder)

}

func TestHttpProvider(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	providerCtx, _ := context.WithCancel(ctx)

	ammoCh := make(chan Ammo, 128)
	provider := &HttpProvider{
		passes:       2,
		ammoFileName: httpTestFilename,
		sink:         ammoCh,
		BaseProvider: NewBaseProvider(
			ammoCh,
			&HttpJSONDecoder{},
			func() interface{} { return &Http{} },
		),
	}
	promise := utils.PromiseCtx(providerCtx, provider.Start)

	ammos := Drain(ctx, provider)
	require.Len(t, ammos, 25*2) // two passes

	httpAmmo, casted := (ammos[2]).(*Http)
	require.True(t, casted, "Ammo should have *Http type")

	assert.Equal(t, "example.org", httpAmmo.Host)
	assert.Equal(t, "/02", httpAmmo.Uri)
	assert.Equal(t, "hello", httpAmmo.Tag)
	assert.Equal(t, "GET", httpAmmo.Method)
	assert.Len(t, httpAmmo.Headers, 4)

	// TODO: add test for decoding error

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

var result Ammo

func BenchmarkJsonDecoder(b *testing.B) {
	f, err := os.Open(httpTestFilename)
	decoder := &HttpJSONDecoder{}
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	jsonDoc, isPrefix, err := r.ReadLine()
	if err != nil || isPrefix {
		b.Fatal(errors.New("Couldn't properly read ammo sample from data file"))
	}
	var a Ammo
	for n := 0; n < b.N; n++ {
		a, _ = decoder.Decode(jsonDoc, &Http{})
	}
	_ = a
}

func BenchmarkJsonDecoderWithPool(b *testing.B) {
	f, err := os.Open(httpTestFilename)
	decoder := &HttpJSONDecoder{}
	if err != nil {
		b.Fatal(err)
	}
	defer f.Close()
	r := bufio.NewReader(f)
	jsonDoc, isPrefix, err := r.ReadLine()
	if err != nil || isPrefix {
		b.Fatal(errors.New("Couldn't properly read ammo sample from data file"))
	}
	var a Ammo
	pool := sync.Pool{
		New: func() interface{} { return &Http{} },
	}
	for n := 0; n < b.N; n++ {
		h := pool.Get().(*Http)
		a, _ = decoder.Decode(jsonDoc, h)
		pool.Put(h)
	}
	_ = a
}
