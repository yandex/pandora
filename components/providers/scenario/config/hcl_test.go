package config

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/lib/pointer"
)

func TestParseHCLFile(t *testing.T) {
	fs := afero.NewOsFs()

	t.Run("http", func(t *testing.T) {
		file, err := fs.Open("../testdata/http_payload.hcl")
		require.NoError(t, err)
		defer file.Close()

		ammoHCL, err := ParseHCLFile(file)
		require.NoError(t, err)

		assert.Len(t, ammoHCL.Scenarios, 2)
		assert.Equal(t, ammoHCL.Scenarios[0], ScenarioHCL{
			Name:           "scenario_name",
			Weight:         pointer.ToInt64(50),
			MinWaitingTime: pointer.ToInt64(10),
			Requests:       []string{"auth_req(1)", "sleep(100)", "list_req(1)", "sleep(100)", "order_req(3)"},
		})
		assert.Equal(t, ammoHCL.Scenarios[1], ScenarioHCL{
			Name:           "scenario_2",
			Weight:         nil,
			MinWaitingTime: nil,
			Requests:       []string{"auth_req(1)", "sleep(100)", "list_req(1)", "sleep(100)", "order_req(2)"},
		})
		assert.Len(t, ammoHCL.VariableSources, 3)
		assert.Equal(t, ammoHCL.VariableSources[2], SourceHCL{
			Name:      "variables",
			Type:      "variables",
			Variables: &(map[string]string{"header": "yandex", "b": "s"})})
	})

	t.Run("grpc", func(t *testing.T) {
		file, err := fs.Open("../testdata/grpc_payload.hcl")
		require.NoError(t, err)
		defer file.Close()

		ammoHCL, err := ParseHCLFile(file)
		require.NoError(t, err)

		assert.Len(t, ammoHCL.Scenarios, 2)
		assert.Equal(t, ammoHCL.Scenarios[0], ScenarioHCL{
			Name:           "scenario_name",
			Weight:         pointer.ToInt64(50),
			MinWaitingTime: pointer.ToInt64(10),
			Requests:       []string{"auth_req(1)", "sleep(100)", "list_req(1)", "sleep(100)", "order_req(3)"},
		})
		assert.Equal(t, ammoHCL.Scenarios[1], ScenarioHCL{
			Name:           "scenario_2",
			Weight:         nil,
			MinWaitingTime: nil,
			Requests:       []string{"auth_req(1)", "sleep(100)", "list_req(1)", "sleep(100)", "order_req(2)"},
		})
		assert.Len(t, ammoHCL.VariableSources, 3)
		assert.Equal(t, ammoHCL.VariableSources[2], SourceHCL{
			Name:      "variables",
			Type:      "variables",
			Variables: &(map[string]string{"header": "yandex", "b": "s"})})
	})
}
