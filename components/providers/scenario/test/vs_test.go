package test

import (
	"testing"

	"github.com/stretchr/testify/require"
	_import "github.com/yandex/pandora/components/providers/scenario/import"
	"github.com/yandex/pandora/components/providers/scenario/vs"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/plugin/pluginconfig"
	"gopkg.in/yaml.v2"
)

func Test_decode_parseVariableSourceCsv(t *testing.T) {
	const exampleVariableSourceYAML = `
src:
 type: "file/csv"
 name: "users_src"
 file: "_files/users.csv"
 ignore_first_line: true
 delimiter: ";"
 fields: [ "user_id", "name" ]
`

	_import.Import(nil)
	testOnce.Do(func() {
		pluginconfig.AddHooks()
	})

	data := make(map[string]any)
	err := yaml.Unmarshal([]byte(exampleVariableSourceYAML), &data)
	require.NoError(t, err)

	out := struct {
		Src vs.VariableSource `yaml:"src"`
	}{}

	err = config.DecodeAndValidate(data, &out)
	require.NoError(t, err)

	csvVS, ok := out.Src.(*vs.VariableSourceCsv)
	require.True(t, ok)
	require.True(t, csvVS.IgnoreFirstLine)
	require.Equal(t, "users_src", csvVS.GetName())
	require.Equal(t, "_files/users.csv", csvVS.File)
	require.Equal(t, []string{"user_id", "name"}, csvVS.Fields)
}

func Test_decode_parseVariableSourceJson(t *testing.T) {
	const exampleVariableSourceJSON = `
src:
 type: "file/json"
 name: "json_src"
 file: "_files/users.json"
`

	_import.Import(nil)
	testOnce.Do(func() {
		pluginconfig.AddHooks()
	})

	data := make(map[string]any)
	err := yaml.Unmarshal([]byte(exampleVariableSourceJSON), &data)
	require.NoError(t, err)

	out := struct {
		Src vs.VariableSource `yaml:"src"`
	}{}

	err = config.DecodeAndValidate(data, &out)
	require.NoError(t, err)

	jsonVS, ok := out.Src.(*vs.VariableSourceJSON)
	require.True(t, ok)
	require.Equal(t, "json_src", jsonVS.GetName())
}
