// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package pluginconfig

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	AddHooks()
}

type TestPlugin interface {
	GetData() string
}

type testPluginImpl struct {
	testPluginConf
}

type testPluginConf struct {
	Data string `validate:"max=20"`
}

func (v *testPluginImpl) GetData() string { return v.Data }

func TestPluginHooks(t *testing.T) {
	dataConf := func(conf interface{}) map[string]interface{} {
		return map[string]interface{}{
			"plugin": conf,
		}
	}
	const pluginName = "test_hook_plugin"
	plugin.Register(reflect.TypeOf((*TestPlugin)(nil)).Elem(), pluginName, func(c testPluginConf) TestPlugin { return &testPluginImpl{c} })

	const expectedData = "expected data"

	validConfig := func() interface{} {
		return dataConf(map[interface{}]interface{}{
			PluginNameKey: pluginName,
			"data":        expectedData,
		})
	}
	invalidConfigs := []map[interface{}]interface{}{
		{},
		{
			PluginNameKey:                  pluginName,
			strings.ToUpper(PluginNameKey): pluginName,
		},
		{
			PluginNameKey: pluginName,
			"data":        expectedData,
			"unused":      "wtf",
		},
		{
			PluginNameKey: pluginName,
			"data":        "invalid because is toooooo looooong",
		},
	}
	testInvalid := func(t *testing.T, data interface{}) {
		for _, tc := range invalidConfigs {
			t.Run(fmt.Sprintf("Invalid conf: %v", tc), func(t *testing.T) {
				err := config.Decode(dataConf(tc), data)
				assert.Error(t, err)
			})
		}
	}

	t.Run("plugin", func(t *testing.T) {
		var data struct {
			Plugin TestPlugin
		}
		err := config.Decode(validConfig(), &data)
		require.NoError(t, err)
		assert.Equal(t, expectedData, data.Plugin.GetData(), expectedData)

		testInvalid(t, data)
	})

	t.Run("factory", func(t *testing.T) {
		var data struct {
			Plugin func() (TestPlugin, error)
		}
		require.True(t, plugin.LookupFactory(plugin.PtrType(&data.Plugin)))
		err := config.Decode(validConfig(), &data)
		require.NoError(t, err)
		plugin, err := data.Plugin()
		require.NoError(t, err)
		assert.Equal(t, expectedData, plugin.GetData(), expectedData)

		testInvalid(t, data)
	})
}
