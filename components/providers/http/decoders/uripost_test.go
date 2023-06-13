package decoders

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/components/providers/base"
	"github.com/yandex/pandora/components/providers/http/config"
)

func Test_uripostDecoder_Scan(t *testing.T) {
	var mustNewAmmo = func(t *testing.T, method string, url string, body []byte, header http.Header, tag string) *base.Ammo {
		ammo, err := base.NewAmmo(method, url, body, header, tag)
		require.NoError(t, err)
		return ammo
	}
	input := `5 /0
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

	decoder := newURIPostDecoder(strings.NewReader(input), config.Config{
		Limit: 8,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()

	wants := []*base.Ammo{
		mustNewAmmo(t, "POST", "/0", []byte("class"), http.Header{"Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "POST", "/1", []byte("class"), http.Header{"A": []string{"b"}, "Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "POST", "/2", []byte("classclass"), http.Header{"Host": []string{"example.com"}, "A": []string{"b"}, "C": []string{"d"}, "Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "POST", "/3", []byte("classclassclass"), http.Header{"Host": []string{"other.net"}, "A": []string{""}, "C": []string{"d"}, "Content-Type": []string{"application/json"}}, "wantTag"),
	}
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
