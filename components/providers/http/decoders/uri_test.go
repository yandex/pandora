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

func Test_uriDecoder_readLine(t *testing.T) {
	var mustNewAmmo = func(t *testing.T, method string, url string, body []byte, header http.Header, tag string) *ammo.Ammo {
		a := ammo.Ammo{}
		err := a.Setup(method, url, body, header, tag)
		require.NoError(t, err)
		return &a
	}

	tests := []struct {
		name                  string
		data                  string
		want                  *ammo.Ammo
		wantErr               bool
		expectedCommonHeaders http.Header
	}{
		{
			name:    "Header line",
			data:    "[Content-Type: application/json]",
			want:    nil,
			wantErr: false,
			expectedCommonHeaders: http.Header{
				"Content-Type": []string{"application/json"},
				"User-Agent":   []string{"TestAgent"},
			},
		},
		{
			name: "Valid URI",
			data: "http://example.com/test",
			want: mustNewAmmo(t, "GET", "http://example.com/test", nil, http.Header{
				"User-Agent":    []string{"TestAgent"},
				"Authorization": []string{"Bearer xxx"},
			}, ""),
			wantErr: false,
			expectedCommonHeaders: http.Header{
				"User-Agent": []string{"TestAgent"},
			},
		},
		{
			name: "URI with tag",
			data: "http://example.com/test tag\n",
			want: mustNewAmmo(t, "GET", "http://example.com/test", nil, http.Header{
				"User-Agent":    []string{"TestAgent"},
				"Authorization": []string{"Bearer xxx"},
			}, "tag"),
			wantErr: false,
			expectedCommonHeaders: http.Header{
				"User-Agent": []string{"TestAgent"},
			},
		},
		{
			name:    "Invalid data",
			data:    "1http://foo.com tag",
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commonHeader := http.Header{"User-Agent": []string{"TestAgent"}}
			decodedConfigHeaders := http.Header{"Authorization": []string{"Bearer xxx"}}

			decoder := newURIDecoder(nil, config.Config{}, decodedConfigHeaders)
			a, err := decoder.readLine(tt.data, commonHeader)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			if tt.want == nil {
				assert.Nil(t, a)
			} else {
				assert.Equal(t, tt.want, a)
			}
			assert.Equal(t, tt.expectedCommonHeaders, commonHeader)
		})
	}
}

const uriDecoderInput = ` /0
[A:b]
/1
[Host : example.com]
[ C : d ]
/2
[A:]
[Host : other.net]

/3
/4 some tag`

func getURIAmmoWants(t *testing.T) []DecodedAmmo {
	var mustNewAmmo = func(t *testing.T, method string, url string, body []byte, header http.Header, tag string) *ammo.Ammo {
		a := ammo.Ammo{}
		err := a.Setup(method, url, body, header, tag)
		require.NoError(t, err)
		return &a
	}
	return []DecodedAmmo{
		mustNewAmmo(t, "GET", "/0", nil, http.Header{"Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "GET", "/1", nil, http.Header{"A": []string{"b"}, "Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "GET", "/2", nil, http.Header{"Host": []string{"example.com"}, "A": []string{"b"}, "C": []string{"d"}, "Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "GET", "/3", nil, http.Header{"Host": []string{"other.net"}, "A": []string{""}, "C": []string{"d"}, "Content-Type": []string{"application/json"}}, ""),
		mustNewAmmo(t, "GET", "/4", nil, http.Header{"Host": []string{"other.net"}, "A": []string{""}, "C": []string{"d"}, "Content-Type": []string{"application/json"}}, "some tag"),
	}
}

func Test_uriDecoder_Scan(t *testing.T) {
	decoder := newURIDecoder(strings.NewReader(uriDecoderInput), config.Config{
		Limit: 10,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	wants := getURIAmmoWants(t)
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

func Test_uriDecoder_LoadAmmo(t *testing.T) {
	decoder := newURIDecoder(strings.NewReader(uriDecoderInput), config.Config{
		Limit: 10,
	}, http.Header{"Content-Type": []string{"application/json"}})

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	wants := getURIAmmoWants(t)

	ammos, err := decoder.LoadAmmo(ctx)
	assert.NoError(t, err)
	assert.Equal(t, wants, ammos)
	assert.Equal(t, decoder.config.Limit, uint(10))
	assert.Equal(t, decoder.config.Passes, uint(0))
}

func Benchmark_uriDecoder_Scan(b *testing.B) {
	decoder := newURIDecoder(
		strings.NewReader(uriDecoderInput), config.Config{},
		http.Header{"Content-Type": []string{"application/json"}},
	)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := decoder.Scan(ctx)
		require.NoError(b, err)
	}
}
