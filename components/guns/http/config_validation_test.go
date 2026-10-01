package phttp

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/plugin/pluginconfig"
	"github.com/yandex/pandora/core/register"
)

func init() {
	pluginconfig.AddHooks()
	register.Gun("test-response-code-http", NewHTTP1GunFactory, DefaultHTTPGunConfig)
	register.Gun("test-response-code-http2", NewHTTP2GunFactory, DefaultHTTP2GunConfig)
	register.Gun("test-response-code-connect", NewConnectGunFactory, DefaultConnectGunConfig)
}

func TestResponseCodeValidatedDuringConfigDecode(t *testing.T) {
	tests := []struct {
		name      string
		gunType   string
		mode      string
		wantError string
	}{
		{name: "grpc on HTTP/1", gunType: "test-response-code-http", mode: "grpc", wantError: "requires gun type http2"},
		{name: "grpc on CONNECT", gunType: "test-response-code-connect", mode: "grpc", wantError: "requires gun type http2"},
		{name: "unknown on HTTP/2", gunType: "test-response-code-http2", mode: "unknown", wantError: "unsupported response-code"},
		{name: "HTTP on HTTP/1", gunType: "test-response-code-http", mode: "http"},
		{name: "HTTP on CONNECT", gunType: "test-response-code-connect", mode: "http"},
		{name: "grpc on HTTP/2", gunType: "test-response-code-http2", mode: "grpc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var decoded struct {
				Gun func() (core.Gun, error) `config:"gun"`
			}
			err := config.Decode(map[string]any{
				"gun": map[string]any{
					"type":          tt.gunType,
					"target":        "localhost:8080",
					"response-code": tt.mode,
				},
			}, &decoded)
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, decoded.Gun)
			gun, err := decoded.Gun()
			require.NoError(t, err)
			require.NotNil(t, gun)
		})
	}
}
