// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/plugin"
)

func TestStringToURLPtrHook(t *testing.T) {
	const (
		validURL   = "http://yandex.ru"
		invalidURL = "http://yandex.ru%%@#$%^&*()%%)(#U@%U)U)##("
	)
	var data struct {
		URLPtr *url.URL `validate:"required"`
	}
	err := DecodeAndValidate(M{"urlptr": validURL}, &data)
	require.NoError(t, err)
	expectedURL, err := url.Parse(validURL)
	require.NoError(t, err)
	assert.Equal(t, data.URLPtr, expectedURL)

	err = DecodeAndValidate(M{"urlptr": invalidURL}, &data)
	assert.Error(t, err)
}

func TestStringToURLHook(t *testing.T) {
	const (
		validURL   = "http://yandex.ru"
		invalidURL = "http://yandex.ru%%@#$%^&*()%%)(#U@%U)U)##("
	)
	var data struct {
		URL url.URL `validate:"required,url"`
	}
	err := DecodeAndValidate(M{"url": validURL}, &data)
	require.NoError(t, err)
	expectedURL, err := url.Parse(validURL)
	require.NoError(t, err)
	assert.Equal(t, data.URL, *expectedURL)

	err = DecodeAndValidate(M{"url": invalidURL}, &data)
	assert.Error(t, err)
}

func TestStringToIPHook(t *testing.T) {
	const (
		validIPv4 = "192.168.0.1"
		validIPv6 = "FF80::1"
		invalidIP = "that is not ip"
	)
	var data struct {
		IP net.IP `validate:"required"`
	}

	err := DecodeAndValidate(M{"ip": validIPv4}, &data)
	require.NoError(t, err)
	expectedIP := net.ParseIP(validIPv4)
	require.NoError(t, err)
	assert.Equal(t, data.IP, expectedIP)

	err = DecodeAndValidate(M{"ip": validIPv6}, &data)
	require.NoError(t, err)
	expectedIP = net.ParseIP(validIPv6)
	require.NoError(t, err)
	assert.Equal(t, data.IP, expectedIP)

	err = DecodeAndValidate(M{"ip": invalidIP}, &data)
	assert.Error(t, err)
}

func TestStringToDataSizeHook(t *testing.T) {
	var data struct {
		Size datasize.ByteSize `validate:"min-size=128b"`
	}

	err := Decode(M{"size": "0"}, &data)
	assert.NoError(t, err)
	assert.Error(t, Validate(data))
	assert.EqualValues(t, 0, data.Size)

	err = Decode(M{"size": "128"}, &data)
	assert.NoError(t, err)
	assert.NoError(t, Validate(data))
	assert.EqualValues(t, 128, data.Size)

	err = Decode(M{"size": "5mb"}, &data)
	assert.NoError(t, err)
	assert.EqualValues(t, 5*datasize.MB, data.Size)

	err = Decode(M{"size": "127KB"}, &data)
	assert.NoError(t, err)
	assert.EqualValues(t, 127*datasize.KB, data.Size)

	err = Decode(M{"size": "Bullshit"}, &data)
	assert.Error(t, err)
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
				err := Decode(dataConf(tc), data)
				assert.Error(t, err)
			})
		}
	}

	t.Run("plugin", func(t *testing.T) {
		var data struct {
			Plugin TestPlugin
		}
		err := Decode(validConfig(), &data)
		require.NoError(t, err)
		assert.Equal(t, expectedData, data.Plugin.GetData(), expectedData)

		testInvalid(t, data)
	})

	t.Run("factory", func(t *testing.T) {
		var data struct {
			Plugin func() (TestPlugin, error)
		}
		require.True(t, plugin.LookupFactory(plugin.PtrType(&data.Plugin)))
		err := Decode(validConfig(), &data)
		require.NoError(t, err)
		plugin, err := data.Plugin()
		require.NoError(t, err)
		assert.Equal(t, expectedData, plugin.GetData(), expectedData)

		testInvalid(t, data)
	})
}
