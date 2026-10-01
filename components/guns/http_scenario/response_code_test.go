package httpscenario

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/plugin/pluginconfig"
)

func init() {
	pluginconfig.AddHooks()
	Import(afero.NewMemMapFs())
}

func TestScenarioResponseCodeValidatedDuringConfigDecode(t *testing.T) {
	for _, gunType := range []string{"http/scenario", "http2/scenario"} {
		for _, mode := range []string{"http", "grpc"} {
			t.Run(gunType+"/"+mode, func(t *testing.T) {
				var decoded struct {
					Gun func() (core.Gun, error) `config:"gun"`
				}
				err := config.Decode(map[string]any{
					"gun": map[string]any{
						"type":          gunType,
						"target":        "localhost:8080",
						"response-code": mode,
					},
				}, &decoded)
				if mode == "grpc" {
					require.ErrorContains(t, err, "response-code")
					return
				}
				require.NoError(t, err)
				gun, err := decoded.Gun()
				require.NoError(t, err)
				require.NotNil(t, gun)
			})
		}
	}
}
