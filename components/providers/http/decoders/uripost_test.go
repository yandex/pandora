package decoders

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders/ammo"
)

const uripostInput = `5 /0
class
[A:b]
5 /1
class
[Host : example.com]
[ C : d ]
10 /2
classclass
[A:]
[Host : other.net]

15 /3 wantTag
classclassclass
`

func getUripostAmmoWants(t *testing.T) []DecodedAmmo {
	var mustNewAmmo = func(t *testing.T, method string, url string, body []byte, header http.Header, tag string) *ammo.Ammo {
		a := ammo.Ammo{}
		err := a.Setup(method, url, body, header, tag)
		require.NoError(t, err)
		return &a
	}
	return []DecodedAmmo{
		mustNewAmmo(t, "POST", "/0", []byte("class"), http.Header{"Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "POST", "/1", []byte("class"), http.Header{"A": []string{"b"}, "Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "POST", "/2", []byte("classclass"), http.Header{"Host": []string{"example.com"}, "A": []string{"b"}, "C": []string{"d"}, "Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "POST", "/3", []byte("classclassclass"), http.Header{"Host": []string{"other.net"}, "A": []string{""}, "C": []string{"d"}, "Content-Type": []string{"application/json"}}, "wantTag"),
	}
}

func Test_uripostDecoder_Scan(t *testing.T) {

	decoder := newURIPostDecoder(strings.NewReader(uripostInput), config.Config{
		Limit: 8,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	wants := getUripostAmmoWants(t)
	for j := 0; j < 2; j++ {
		for i, want := range wants {
			ammo, err := decoder.Scan(ctx)
			assert.NoError(t, err, "iteration %d-%d", j, i)
			assert.Equal(t, want, ammo, "iteration %d-%d", j, i)
		}
	}

	_, err := decoder.Scan(ctx)
	assert.Equal(t, err, ErrAmmoLimit)
	assert.Equal(t, decoder.ammoNum, uint(len(wants)*2))
	assert.Equal(t, decoder.passNum, uint(1))
}

func Test_uripostDecoder_LoadAmmo(t *testing.T) {
	decoder := newURIPostDecoder(strings.NewReader(uripostInput), config.Config{
		Limit: 8,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	wants := getUripostAmmoWants(t)

	ammos, err := decoder.LoadAmmo(ctx)
	assert.NoError(t, err)
	assert.Equal(t, wants, ammos)
	assert.Equal(t, decoder.config.Limit, uint(8))
	assert.Equal(t, decoder.config.Passes, uint(0))
}
