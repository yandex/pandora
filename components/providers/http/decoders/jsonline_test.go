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

func Test_jsonlineDecoder_Scan(t *testing.T) {
	var mustNewAmmo = func(t *testing.T, method string, url string, body []byte, header http.Header, tag string) *ammo.Ammo {
		a := ammo.Ammo{}
		err := a.Setup(method, url, body, header, tag)
		require.NoError(t, err)
		return &a
	}

	input := `{"host": "ya.net", "method": "GET", "uri": "/?sleep=100", "tag": "sleep1", "headers": {"User-agent": "Tank", "Connection": "close"}}
{"host": "ya.net", "method": "POST", "uri": "/?sleep=200", "tag": "sleep2", "headers": {"User-agent": "Tank", "Connection": "close"}, "body": "body_data"}


`

	decoder := newJsonlineDecoder(strings.NewReader(input), config.Config{
		Limit: 4,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	wants := []*ammo.Ammo{
		mustNewAmmo(t,
			"GET",
			"http://ya.net/?sleep=100",
			nil,
			http.Header{"Connection": []string{"close"}, "Content-Type": []string{"application/json"}, "User-Agent": []string{"Tank"}},
			"sleep1",
		),
		mustNewAmmo(t,
			"POST",
			"http://ya.net/?sleep=200",
			[]byte("body_data"),
			http.Header{"Connection": []string{"close"}, "Content-Type": []string{"application/json"}, "User-Agent": []string{"Tank"}},
			"sleep2",
		),
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
