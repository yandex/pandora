// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterValid(t *testing.T) {
	testCases := []struct {
		description              string
		newPluginImpl            interface{}
		newDefaultConfigOptional []interface{}
	}{
		{"Return Impl", func() *testPluginImpl { return nil }, nil},
		{"Return Interface", func() TestPlugin { return nil }, nil},
		{"Super interface", func() interface {
			io.Writer
			TestPlugin
		} {
			return nil
		}, nil},
		{"Struct config", func(pluginImplConfig) TestPlugin { return nil }, nil},
		{"Struct ptr config", func(*pluginImplConfig) TestPlugin { return nil }, nil},
		//{"String map config", func(map[string]interface{}) TestPlugin { return nil }, nil},
		{"Default config", func(*pluginImplConfig) TestPlugin { return nil }, []interface{}{func() *pluginImplConfig { return nil }}},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.NotPanics(t, func() {
				newRegistry().testRegister(tc.newPluginImpl, tc.newDefaultConfigOptional...)
			})
		})
	}
}

func TestRegisterInvalid(t *testing.T) {
	testCases := []struct {
		description              string
		newPluginImpl            interface{}
		newDefaultConfigOptional []interface{}
	}{
		{"Return not impl", func() testPluginImpl { panic("") }, nil},
		{"Invalid config", func(int) TestPlugin { return nil }, nil},
		{"To many args", func(_, _ pluginImplConfig) TestPlugin { return nil }, nil},
		{"Default without config", func() TestPlugin { return nil }, []interface{}{func() *pluginImplConfig { return nil }}},
		{"Extra deafult config", func(*pluginImplConfig) TestPlugin { return nil }, []interface{}{func() *pluginImplConfig { return nil }, 0}},
		{"Invalid default config", func(pluginImplConfig) TestPlugin { return nil }, []interface{}{func() *pluginImplConfig { return nil }}},
		{"Default config accepts args", func(*pluginImplConfig) TestPlugin { return nil }, []interface{}{func(int) *pluginImplConfig { return nil }}},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			defer func() {
				r := recover()
				require.NotNil(t, r)
				assert.Contains(t, r, "expectation failed")
			}()
			newRegistry().testRegister(tc.newPluginImpl, tc.newDefaultConfigOptional...)
		})
	}
}

func TestNew(t *testing.T) {
	var r registry

	fillConf := func(conf interface{}) error {
		return mapstructure.Decode(map[string]interface{}{"Value": "conf"}, conf)
	}
	tests := []struct {
		desc string
		fn   func(t *testing.T)
	}{
		{"no conf", func(t *testing.T) {
			r.testRegister(newPlugin)
			assert.Equal(t, r.testNewOk(t), "init")
		}},
		{"no conf, fill conf error", func(t *testing.T) {
			r.testRegister(newPlugin)
			expectedErr := errors.New("fill conf err")
			_, err := r.testNew(func(_ interface{}) error { return expectedErr })
			assert.Equal(t, expectedErr, err)
		}},
		{"no default", func(t *testing.T) {
			r.testRegister(func(c pluginImplConfig) *testPluginImpl { return &testPluginImpl{c.Value} })
			assert.Equal(t, r.testNewOk(t), "")
		}},
		{"default", func(t *testing.T) {
			r.testRegister(newPluginFromConf, newPluginDefaultConf)
			assert.Equal(t, r.testNewOk(t), "default")
		}},
		{"fill conf default", func(t *testing.T) {
			r.testRegister(newPluginFromConf, newPluginDefaultConf)
			assert.Equal(t, "conf", r.testNewOk(t, fillConf))
		}},
		{"fill conf no default", func(t *testing.T) {
			r.testRegister(newPluginFromConf)
			assert.Equal(t, "conf", r.testNewOk(t, fillConf))
		}},
		{"fill ptr conf no default", func(t *testing.T) {
			r.testRegister(func(c *pluginImplConfig) *testPluginImpl { return &testPluginImpl{c.Value} })
			assert.Equal(t, "conf", r.testNewOk(t, fillConf))
		}},
	}
	for _, test := range tests {
		r = newRegistry()
		t.Run(test.desc, test.fn)
	}
}

func (r registry) testRegister(newPluginImpl interface{}, newDefaultConfigOptional ...interface{}) {
	r.Register(pluginType(), "test_name", newPluginImpl, newDefaultConfigOptional...)
}

func (r registry) testNew(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	return r.New(pluginType(), "test_name", fillConfOptional...)
}

func (r registry) testNewOk(t *testing.T, fillConfOptional ...func(conf interface{}) error) (pluginVal string) {
	plugin, err := r.New(pluginType(), "test_name", fillConfOptional...)
	require.NoError(t, err)
	return plugin.(*testPluginImpl).Value
}

type TestPlugin interface {
	DoSomething()
}

func pluginType() reflect.Type   { return reflect.TypeOf((*TestPlugin)(nil)).Elem() }
func newPlugin() *testPluginImpl { return &testPluginImpl{Value: "init"} }

type testPluginImpl struct{ Value string }

func (p *testPluginImpl) DoSomething() {}

var _ TestPlugin = (*testPluginImpl)(nil)

type pluginImplConfig struct{ Value string }

func newPluginFromConf(c pluginImplConfig) *testPluginImpl { return &testPluginImpl{c.Value} }
func newPluginDefaultConf() pluginImplConfig               { return pluginImplConfig{"default"} }

func TestMapstructureDecode(t *testing.T) {
	r := newRegistry()
	const nameKey = "type"

	var hook mapstructure.DecodeHookFunc
	decode := func(input, result interface{}) error {
		decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
			DecodeHook:  hook,
			ErrorUnused: true,
			Result:      result,
		})
		if err != nil {
			return err
		}
		return decoder.Decode(input)
	}
	hook = mapstructure.ComposeDecodeHookFunc(
		func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
			if !r.Lookup(to) {
				return data, nil
			}
			// NOTE: could be map[interface{}]interface{} here
			input := data.(map[string]interface{})
			// NOTE: should be case insensitive
			pluginName := input[nameKey].(string)
			delete(input, nameKey)
			return r.New(to, pluginName, func(conf interface{}) error {
				// NOTE: error, if conf has "type" field
				return decode(input, conf)
			})
		})

	r.Register(pluginType(), "my-plugin", newPluginFromConf, newPluginDefaultConf)
	input := map[string]interface{}{
		"plugin": map[string]interface{}{
			nameKey: "my-plugin",
			"value": "conf",
		},
	}
	type Config struct {
		Plugin TestPlugin
	}
	var conf Config
	err := decode(input, &conf)
	require.NoError(t, err)
	assert.Equal(t, "conf", conf.Plugin.(*testPluginImpl).Value)
}
