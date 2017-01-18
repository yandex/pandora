// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MLP 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/facebookgo/stackerr"
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
		{"return impl", func() *testPluginImpl { return nil }, nil},
		{"return interface", func() TestPlugin { return nil }, nil},
		{"super interface", func() interface {
			io.Writer
			TestPlugin
		} {
			return nil
		}, nil},
		{"struct config", func(pluginImplConfig) TestPlugin { return nil }, nil},
		{"struct ptr config", func(*pluginImplConfig) TestPlugin { return nil }, nil},
		{"default config", func(*pluginImplConfig) TestPlugin { return nil }, []interface{}{func() *pluginImplConfig { return nil }}},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			assert.NotPanics(t, func() {
				newTypeRegistry().testRegister(tc.newPluginImpl, tc.newDefaultConfigOptional...)
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
		{"return not impl", func() testPluginImpl { panic("") }, nil},
		{"invalid config type", func(int) TestPlugin { return nil }, nil},
		{"invalid config ptr type", func(*int) TestPlugin { return nil }, nil},
		{"to many args", func(_, _ pluginImplConfig) TestPlugin { return nil }, nil},
		{"default without config", func() TestPlugin { return nil }, []interface{}{func() *pluginImplConfig { return nil }}},
		{"extra deafult config", func(*pluginImplConfig) TestPlugin { return nil }, []interface{}{func() *pluginImplConfig { return nil }, 0}},
		{"invalid default config", func(pluginImplConfig) TestPlugin { return nil }, []interface{}{func() *pluginImplConfig { return nil }}},
		{"default config accepts args", func(*pluginImplConfig) TestPlugin { return nil }, []interface{}{func(int) *pluginImplConfig { return nil }}},
	}
	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			defer assertExpectationFailed(t)
			newTypeRegistry().testRegister(tc.newPluginImpl, tc.newDefaultConfigOptional...)
		})
	}
}

func TestRegisterNameCollisionPanics(t *testing.T) {
	r := newTypeRegistry()
	r.testRegister(newPlugin)
	defer assertExpectationFailed(t)
	r.testRegister(newPlugin)
}

func TestLookup(t *testing.T) {
	r := newTypeRegistry()
	r.testRegister(newPlugin)
	assert.True(t, r.Lookup(pluginType()))
	assert.False(t, r.Lookup(reflect.TypeOf(0)))
	assert.False(t, r.Lookup(reflect.TypeOf(&testPluginImpl{})))
	assert.False(t, r.Lookup(reflect.TypeOf((*io.Writer)(nil)).Elem()))
}

func TestNew(t *testing.T) {
	var r typeRegistry

	type New func(r typeRegistry, fillConfOptional ...func(conf interface{}) error) (interface{}, error)
	var testNew New
	testNewOk := func(fillConfOptional ...func(conf interface{}) error) (pluginVal string) {
		plugin, err := testNew(r, fillConfOptional...)
		require.NoError(t, err)
		return plugin.(*testPluginImpl).Value
	}

	tests := []struct {
		desc string
		fn   func(t *testing.T)
	}{
		{"no conf", func(t *testing.T) {
			r.testRegister(newPlugin)
			assert.Equal(t, testNewOk(), testInitValue)
		}},
		{"nil error", func(t *testing.T) {
			r.testRegister(func() (TestPlugin, error) {
				return newPlugin(), nil
			})
			assert.Equal(t, testNewOk(), testInitValue)
		}},
		{"non-nil error", func(t *testing.T) {
			expectedErr := errors.New("fill conf err")
			r.testRegister(func() (TestPlugin, error) {
				return nil, expectedErr
			})
			_, err := testNew(r)
			require.Error(t, err)
			errs := stackerr.Underlying(err)
			err = errs[len(errs)-1]
			assert.Equal(t, expectedErr, err)
		}},
		{"no conf, fill conf error", func(t *testing.T) {
			r.testRegister(newPlugin)
			expectedErr := errors.New("fill conf err")
			_, err := testNew(r, func(_ interface{}) error { return expectedErr })
			assert.Equal(t, expectedErr, err)
		}},
		{"no default", func(t *testing.T) {
			r.testRegister(func(c pluginImplConfig) *testPluginImpl { return &testPluginImpl{c.Value} })
			assert.Equal(t, testNewOk(), "")
		}},
		{"default", func(t *testing.T) {
			r.testRegister(newPluginConf, newPluginDefaultConf)
			assert.Equal(t, testNewOk(), testDefaultValue)
		}},
		{"fill conf default", func(t *testing.T) {
			r.testRegister(newPluginConf, newPluginDefaultConf)
			assert.Equal(t, "conf", testNewOk(fillConf))
		}},
		{"fill conf no default", func(t *testing.T) {
			r.testRegister(newPluginConf)
			assert.Equal(t, "conf", testNewOk(fillConf))
		}},
		{"fill ptr conf no default", func(t *testing.T) {
			r.testRegister(newPluginPtrConf)
			assert.Equal(t, "conf", testNewOk(fillConf))
		}},
		{"no default ptr conf not nil", func(t *testing.T) {
			r.testRegister(newPluginPtrConf)
			assert.Equal(t, "", testNewOk())
		}},
		{"nil default, conf not nil", func(t *testing.T) {
			r.testRegister(newPluginPtrConf, func() *pluginImplConfig { return nil })
			assert.Equal(t, "", testNewOk())
		}},
		{"fill nil default", func(t *testing.T) {
			r.testRegister(newPluginPtrConf, func() *pluginImplConfig { return nil })
			assert.Equal(t, "conf", testNewOk(fillConf))
		}},
		{"more than one fill conf panics", func(t *testing.T) {
			r.testRegister(newPluginPtrConf)
			defer assertExpectationFailed(t)
			testNew(r, fillConf, fillConf)
		}},
	}

	for _, suite := range []struct {
		new  New
		desc string
	}{
		{typeRegistry.testNew, "New"},
		{typeRegistry.testNewFactory, "NewFactory"},
	} {
		testNew = suite.new
		for _, test := range tests {
			r = newTypeRegistry()
			t.Run(fmt.Sprintf("%s %s", suite.desc, test.desc), test.fn)
		}
	}

}

// Test typical usage.
func TestMapstructureDecode(t *testing.T) {
	r := newTypeRegistry()
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
			// NOTE: could be map[interface{}]interface{} here.
			input := data.(map[string]interface{})
			// NOTE: should be case insensitive behaviour.
			pluginName := input[nameKey].(string)
			delete(input, nameKey)
			return r.New(to, pluginName, func(conf interface{}) error {
				// NOTE: should error, if conf has "type" field.
				return decode(input, conf)
			})
		})

	r.Register(pluginType(), "my-plugin", newPluginConf, newPluginDefaultConf)
	input := map[string]interface{}{
		"plugin": map[string]interface{}{
			nameKey: "my-plugin",
			"value": testConfValue,
		},
	}
	type Config struct {
		Plugin TestPlugin
	}
	var conf Config
	err := decode(input, &conf)
	require.NoError(t, err)
	assert.Equal(t, testConfValue, conf.Plugin.(*testPluginImpl).Value)
}

const (
	testPluginName   = "test_name"
	testConfValue    = "conf"
	testDefaultValue = "default"
	testInitValue    = "init"
)

func (r typeRegistry) testRegister(newPluginImpl interface{}, newDefaultConfigOptional ...interface{}) {
	r.Register(pluginType(), testPluginName, newPluginImpl, newDefaultConfigOptional...)
}

func (r typeRegistry) testNew(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	return r.New(pluginType(), testPluginName, fillConfOptional...)
}

func (r typeRegistry) testNewFactory(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	factory, err := r.NewFactory(pluginFactoryType(), testPluginName, fillConfOptional...)
	if err != nil {
		return
	}
	typedFactory := factory.(func() (TestPlugin, error))
	return typedFactory()
}

type TestPlugin interface {
	DoSomething()
}

func pluginType() reflect.Type        { return reflect.TypeOf((*TestPlugin)(nil)).Elem() }
func pluginFactoryType() reflect.Type { return reflect.TypeOf(func() (TestPlugin, error) { panic("") }) }
func newPlugin() *testPluginImpl      { return &testPluginImpl{Value: testInitValue} }

type testPluginImpl struct{ Value string }

func (p *testPluginImpl) DoSomething() {}

var _ TestPlugin = (*testPluginImpl)(nil)

type pluginImplConfig struct{ Value string }

func newPluginConf(c pluginImplConfig) *testPluginImpl { return &testPluginImpl{c.Value} }
func newPluginDefaultConf() pluginImplConfig           { return pluginImplConfig{testDefaultValue} }
func newPluginPtrConf(c *pluginImplConfig) *testPluginImpl {
	return &testPluginImpl{c.Value}
}

func fillConf(conf interface{}) error {
	return mapstructure.Decode(map[string]interface{}{"Value": "conf"}, conf)
}

func assertExpectationFailed(t *testing.T) {
	r := recover()
	require.NotNil(t, r)
	assert.Contains(t, r, "expectation failed")
}
