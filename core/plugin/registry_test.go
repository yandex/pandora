// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	stderrors "errors"
	"fmt"
	"io"
	"reflect"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
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
		{"return interface", func() testPluginInterface { return nil }, nil},
		{"super interface", func() interface {
			io.Writer
			testPluginInterface
		} {
			return nil
		}, nil},
		{"struct config", func(testPluginImplConfig) testPluginInterface { return nil }, nil},
		{"struct ptr config", func(*testPluginImplConfig) testPluginInterface { return nil }, nil},
		{"default config", func(*testPluginImplConfig) testPluginInterface { return nil }, []interface{}{func() *testPluginImplConfig { return nil }}},
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
		{"invalid config type", func(int) testPluginInterface { return nil }, nil},
		{"invalid config ptr type", func(*int) testPluginInterface { return nil }, nil},
		{"to many args", func(_, _ testPluginImplConfig) testPluginInterface { return nil }, nil},
		{"default without config", func() testPluginInterface { return nil }, []interface{}{func() *testPluginImplConfig { return nil }}},
		{"extra deafult config", func(*testPluginImplConfig) testPluginInterface { return nil }, []interface{}{func() *testPluginImplConfig { return nil }, 0}},
		{"invalid default config", func(testPluginImplConfig) testPluginInterface { return nil }, []interface{}{func() *testPluginImplConfig { return nil }}},
		{"default config accepts args", func(*testPluginImplConfig) testPluginInterface { return nil }, []interface{}{func(int) *testPluginImplConfig { return nil }}},
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
	r.testRegister(newTestPlugin)
	defer assertExpectationFailed(t)
	r.testRegister(newTestPlugin)
}

func TestLookup(t *testing.T) {
	r := newTypeRegistry()
	r.testRegister(newTestPlugin)
	assert.True(t, r.Lookup(testPluginType()))
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
			r.testRegister(newTestPlugin)
			assert.Equal(t, testNewOk(), testInitValue)
		}},
		{"nil error", func(t *testing.T) {
			r.testRegister(func() (testPluginInterface, error) {
				return newTestPlugin(), nil
			})
			assert.Equal(t, testNewOk(), testInitValue)
		}},
		{"non-nil error", func(t *testing.T) {
			expectedErr := stderrors.New("fill conf err")
			r.testRegister(func() (testPluginInterface, error) {
				return nil, expectedErr
			})
			_, err := testNew(r)
			require.Error(t, err)
			err = errors.Cause(err)
			assert.Equal(t, expectedErr, err)
		}},
		{"no conf, fill conf error", func(t *testing.T) {
			r.testRegister(newTestPlugin)
			expectedErr := stderrors.New("fill conf err")
			_, err := testNew(r, func(_ interface{}) error { return expectedErr })
			assert.Equal(t, expectedErr, err)
		}},
		{"no default", func(t *testing.T) {
			r.testRegister(func(c testPluginImplConfig) *testPluginImpl { return &testPluginImpl{c.Value} })
			assert.Equal(t, testNewOk(), "")
		}},
		{"default", func(t *testing.T) {
			r.testRegister(newTestPluginConf, newTestPluginDefaultConf)
			assert.Equal(t, testNewOk(), testDefaultValue)
		}},
		{"fill conf default", func(t *testing.T) {
			r.testRegister(newTestPluginConf, newTestPluginDefaultConf)
			assert.Equal(t, "conf", testNewOk(fillTestPluginConf))
		}},
		{"fill conf no default", func(t *testing.T) {
			r.testRegister(newTestPluginConf)
			assert.Equal(t, "conf", testNewOk(fillTestPluginConf))
		}},
		{"fill ptr conf no default", func(t *testing.T) {
			r.testRegister(newTestPluginPtrConf)
			assert.Equal(t, "conf", testNewOk(fillTestPluginConf))
		}},
		{"no default ptr conf not nil", func(t *testing.T) {
			r.testRegister(newTestPluginPtrConf)
			assert.Equal(t, "", testNewOk())
		}},
		{"nil default, conf not nil", func(t *testing.T) {
			r.testRegister(newTestPluginPtrConf, func() *testPluginImplConfig { return nil })
			assert.Equal(t, "", testNewOk())
		}},
		{"fill nil default", func(t *testing.T) {
			r.testRegister(newTestPluginPtrConf, func() *testPluginImplConfig { return nil })
			assert.Equal(t, "conf", testNewOk(fillTestPluginConf))
		}},
		{"more than one fill conf panics", func(t *testing.T) {
			r.testRegister(newTestPluginPtrConf)
			defer assertExpectationFailed(t)
			testNew(r, fillTestPluginConf, fillTestPluginConf)
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

	r.Register(testPluginType(), "my-plugin", newTestPluginConf, newTestPluginDefaultConf)
	input := map[string]interface{}{
		"plugin": map[string]interface{}{
			nameKey: "my-plugin",
			"value": testConfValue,
		},
	}
	type Config struct {
		Plugin testPluginInterface
	}
	var conf Config
	err := decode(input, &conf)
	require.NoError(t, err)
	assert.Equal(t, testConfValue, conf.Plugin.(*testPluginImpl).Value)
}

func assertExpectationFailed(t *testing.T) {
	r := recover()
	require.NotNil(t, r)
	assert.Contains(t, r, "expectation failed")
}
