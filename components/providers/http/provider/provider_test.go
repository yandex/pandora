package provider

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/components/providers/http/config"
	"github.com/yandex/pandora/components/providers/http/decoders"
	"github.com/yandex/pandora/components/providers/http/decoders/ammo"
)

func TestProvider_runPreloaded(t *testing.T) {
	var mustNewAmmo = func(t *testing.T, method string, url string, body []byte, header http.Header, tag string) *ammo.Ammo {
		a := ammo.Ammo{}
		err := a.Setup(method, url, body, header, tag)
		require.NoError(t, err)
		return &a
	}

	tests := []struct {
		name           string
		contextTimeout time.Duration
		cfg            config.Config
		wantAmmos      int
		wantErr        string
	}{
		{
			name:           "ammo limit",
			contextTimeout: time.Second,
			cfg: config.Config{
				Passes: 0,
				Limit:  5,
			},
			wantAmmos: 5,
			wantErr:   decoders.ErrAmmoLimit.Error(),
		},
		{
			name:           "pass limit",
			contextTimeout: time.Second,
			cfg: config.Config{
				Passes: 1,
				Limit:  9,
			},
			wantAmmos: 3,
			wantErr:   decoders.ErrPassLimit.Error(),
		},
		{
			name:           "context deadline exceeded",
			contextTimeout: 15 * time.Millisecond,
			cfg: config.Config{
				Passes: 1,
				Limit:  9,
			},
			wantAmmos: 2,
			wantErr:   "error from context: context deadline exceeded",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &Provider{
				ammos: []decoders.DecodedAmmo{
					mustNewAmmo(t, "GET", "get", nil, nil, "tagget"),
					mustNewAmmo(t, "POST", "post", nil, nil, "tagpost"),
					mustNewAmmo(t, "PUT", "put", nil, nil, "tagput"),
				},
				Config: tt.cfg,
				Sink:   make(chan decoders.DecodedAmmo),
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.contextTimeout)
			defer cancel()

			i := 0
			cl := make(chan struct{})
			go func() {
				defer close(cl)
				for a := range provider.Sink {
					req, err := a.BuildRequest()
					assert.NoError(t, err)
					switch i % 3 {
					case 0:
						require.Equal(t, "GET", req.Method)
					case 1:
						require.Equal(t, "POST", req.Method)
					case 2:
						require.Equal(t, "PUT", req.Method)
					}
					i++
					time.Sleep(10 * time.Millisecond) // for test context deadline exceeded
				}
			}()

			err := provider.runPreloaded(ctx)
			assert.EqualError(t, err, tt.wantErr)

			close(provider.Sink)
			<-cl
			assert.Equal(t, tt.wantAmmos, i)
		})
	}

}

func TestLoadAmmo_ChosenTags(t *testing.T) {
	var mustNewAmmo = func(t *testing.T, method string, url string, body []byte, header http.Header, tag string) *ammo.Ammo {
		a := ammo.Ammo{}
		err := a.Setup(method, url, body, header, tag)
		require.NoError(t, err)
		return &a
	}
	ammo1 := mustNewAmmo(t, "GET", "", nil, make(http.Header), "tag1")
	ammo2 := mustNewAmmo(t, "PUT", "", nil, make(http.Header), "tag2")
	ammo3 := mustNewAmmo(t, "POST", "", nil, make(http.Header), "tag3")

	decoder := decoders.NewMockDecoder(t)
	decoder.On("LoadAmmo", mock.Anything).Return([]decoders.DecodedAmmo{ammo1, ammo2, ammo3}, nil)

	provider := &Provider{
		Decoder: decoder,
		ammos:   nil,
		Config:  config.Config{ChosenCases: []string{"tag1", "tag3"}},
	}

	err := provider.loadAmmo(context.Background())
	assert.NoError(t, err)

	expectedAmmos := []decoders.DecodedAmmo{ammo1, ammo3}

	assert.Len(t, provider.ammos, len(expectedAmmos))
	assert.Equal(t, provider.ammos, expectedAmmos)
}
