// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"io"
	"reflect"
	"testing"

	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaultConfigContainerExpectationFail(t *testing.T) {
	tests := []struct {
		name                     string
		constructor              any
		newDefaultConfigOptional []any
	}{
		{
			name:        "invalid type",
			constructor: func(int) ptestPlugin { return nil },
		},
		{
			name:        "invalid ptr type",
			constructor: func(*int) ptestPlugin { return nil },
		},
		{
			name:        "to many args",
			constructor: func(_, _ ptestConfig) ptestPlugin { return nil },
		},
		{
			name:                     "default without config",
			constructor:              func() ptestPlugin { return nil },
			newDefaultConfigOptional: []any{func() *ptestConfig { return nil }}},
		{
			name:                     "invalid default config",
			constructor:              func(ptestConfig) ptestPlugin { return nil },
			newDefaultConfigOptional: []any{func() *ptestConfig { return nil }}},
		{
			name:                     "default config accepts args",
			constructor:              func(*ptestConfig) ptestPlugin { return nil },
			newDefaultConfigOptional: []any{func(int) *ptestConfig { return nil }},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newDefaultConfig := getNewDefaultConfig(tt.newDefaultConfigOptional)
			defer recoverExpectationFail(t)
			newDefaultConfigContainer(reflect.TypeOf(tt.constructor), newDefaultConfig)
		})
	}
}

func TestNewDefaultConfigContainerExpectationOk(t *testing.T) {
	tests := []struct {
		name                     string
		constructor              any
		newDefaultConfigOptional []any
	}{

		{
			name:        "no default config",
			constructor: ptestNewConf},
		{
			name:        "no default ptr config",
			constructor: ptestNewPtrConf},
		{
			name:                     "default config",
			constructor:              ptestNewConf,
			newDefaultConfigOptional: []any{ptestDefaultConf}},
		{
			name:                     "default ptr config",
			constructor:              ptestNewPtrConf,
			newDefaultConfigOptional: []any{ptestNewDefaultPtrConf},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newDefaultConfig := getNewDefaultConfig(tt.newDefaultConfigOptional)
			container := newDefaultConfigContainer(reflect.TypeOf(tt.constructor), newDefaultConfig)
			conf, err := container.Get(ptestFillConf)
			assert.NoError(t, err)
			assert.Len(t, conf, 1)
			ptestExpectConfigValue(t, conf[0].Interface(), ptestFilledValue)
		})
	}
}

// new default config container fill no config failed
func TestNewDefault(t *testing.T) {
	container := newDefaultConfigContainer(ptestNewErrType(), nil)
	_, err := container.Get(ptestFillConf)
	assert.Error(t, err)
}

func TestRegistry(t *testing.T) {
	t.Run("register name collision panics", func(t *testing.T) {
		r := NewRegistry()
		r.ptestRegister(ptestNewImpl)
		defer recoverExpectationFail(t)
		r.ptestRegister(ptestNewImpl)
	})

	t.Run("lookup", func(t *testing.T) {
		r := NewRegistry()
		r.ptestRegister(ptestNewImpl)
		assert.True(t, r.Lookup(ptestType()))
		assert.False(t, r.Lookup(reflect.TypeOf(0)))
		assert.False(t, r.Lookup(reflect.TypeOf(&ptestImpl{})))
		assert.False(t, r.Lookup(reflect.TypeOf((*io.Writer)(nil)).Elem()))
	})

	t.Run("lookup factory", func(t *testing.T) {
		r := NewRegistry()
		r.ptestRegister(ptestNewImpl)
		assert.True(t, r.LookupFactory(ptestNewType()))
		assert.True(t, r.LookupFactory(ptestNewErrType()))

		assert.False(t, r.LookupFactory(reflect.TypeOf(0)))
		assert.False(t, r.LookupFactory(reflect.TypeOf(&ptestImpl{})))
		assert.False(t, r.LookupFactory(reflect.TypeOf((*io.Writer)(nil)).Elem()))
	})
}

func TestNew(t *testing.T) {
	type New func(r *Registry, fillConfOptional ...func(conf interface{}) error) (interface{}, error)

	testNewOk := func(t *testing.T, r *Registry, testNew New, fillConfOptional ...func(conf interface{}) error) (pluginVal string) {
		plugin, err := testNew(r, fillConfOptional...)
		require.NoError(t, err)
		return plugin.(*ptestImpl).Value
	}

	tests := []struct {
		name string

		assert func(t *testing.T, r *Registry, testNew New)
	}{
		{
			name: "plugin constructor. no conf",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewImpl)
				got := testNewOk(t, r, testNew)
				assert.Equal(t, ptestInitValue, got)
			},
		},
		{
			name: "plugin conf: nil error",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewErr)
				got := testNewOk(t, r, testNew)
				assert.Equal(t, ptestInitValue, got)
			},
		},
		{
			name: "plugin conf: non-nil error",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewErrFailing)
				_, err := testNew(r)
				assert.Error(t, err)
				assert.ErrorIs(t, err, ptestCreateFailedErr)
			},
		},
		{
			name: "plugin conf: no conf, fill conf error",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewImpl)
				expectedErr := errors.New("fill conf err")
				_, err := testNew(r, func(_ interface{}) error { return expectedErr })
				assert.ErrorIs(t, err, expectedErr)
			},
		},
		{
			name: "plugin conf: no default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewConf)
				got := testNewOk(t, r, testNew)
				assert.Equal(t, "", got)
			},
		},
		{
			name: "plugin conf: default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewConf, ptestDefaultConf)
				got := testNewOk(t, r, testNew)
				assert.Equal(t, ptestDefaultValue, got)
			},
		},
		{
			name: "plugin conf: fill conf default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewConf, ptestDefaultConf)
				got := testNewOk(t, r, testNew, ptestFillConf)
				assert.Equal(t, ptestFilledValue, got)
			},
		},
		{
			name: "plugin conf: fill conf no default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewConf)
				got := testNewOk(t, r, testNew, ptestFillConf)
				assert.Equal(t, ptestFilledValue, got)
			},
		},
		{
			name: "plugin conf: fill ptr conf no default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewPtrConf)
				got := testNewOk(t, r, testNew, ptestFillConf)
				assert.Equal(t, ptestFilledValue, got)
			},
		},
		{
			name: "plugin conf: no default ptr conf not nil",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewPtrConf)
				got := testNewOk(t, r, testNew)
				assert.Equal(t, "", got)
			},
		},
		{
			name: "plugin conf: nil default, conf not nil",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewPtrConf, func() *ptestConfig { return nil })
				got := testNewOk(t, r, testNew)
				assert.Equal(t, "", got)
			},
		},
		{
			name: "plugin conf: fill nil default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewPtrConf, func() *ptestConfig { return nil })
				got := testNewOk(t, r, testNew, ptestFillConf)
				assert.Equal(t, ptestFilledValue, got)
			},
		},
		{
			name: "plugin conf: more than one fill conf panics",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewPtrConf)
				defer recoverExpectationFail(t)
				testNew(r, ptestFillConf, ptestFillConf)
			},
		},
		{
			name: "factory constructor; no conf",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactory)
				got := testNewOk(t, r, testNew)
				assert.Equal(t, ptestInitValue, got)
			},
		},
		{
			name: "factory constructor; nil error",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(func() (ptestPlugin, error) {
					return ptestNewImpl(), nil
				})
				got := testNewOk(t, r, testNew)
				assert.Equal(t, ptestInitValue, got)
			},
		},
		{
			name: "factory constructor; non-nil error",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryFactoryErrFailing)
				_, err := testNew(r)
				assert.Error(t, err)
				assert.ErrorIs(t, err, ptestCreateFailedErr)
			},
		},
		{
			name: "factory constructor; no conf, fill conf error",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactory)
				expectedErr := errors.New("fill conf err")
				_, err := testNew(r, func(_ interface{}) error { return expectedErr })
				assert.ErrorIs(t, err, expectedErr)
			},
		},
		{
			name: "factory constructor; no default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryConf)
				got := testNewOk(t, r, testNew)
				assert.Equal(t, "", got)
			},
		},
		{
			name: "factory constructor; default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryConf, ptestDefaultConf)
				got := testNewOk(t, r, testNew)
				assert.Equal(t, ptestDefaultValue, got)
			},
		},
		{
			name: "factory constructor; fill conf default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryConf, ptestDefaultConf)
				got := testNewOk(t, r, testNew, ptestFillConf)
				assert.Equal(t, ptestFilledValue, got)
			},
		},
		{
			name: "factory constructor; fill conf no default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryConf)
				got := testNewOk(t, r, testNew, ptestFillConf)
				assert.Equal(t, ptestFilledValue, got)
			},
		},
		{
			name: "factory constructor; fill ptr conf no default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryPtrConf)
				got := testNewOk(t, r, testNew, ptestFillConf)
				assert.Equal(t, ptestFilledValue, got)
			},
		},
		{
			name: "factory constructor; no default ptr conf not nil",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryPtrConf)
				got := testNewOk(t, r, testNew)
				assert.Equal(t, "", got)
			},
		},
		{
			name: "factory constructor; nil default, conf not nil",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryPtrConf, func() *ptestConfig { return nil })
				got := testNewOk(t, r, testNew)
				assert.Equal(t, "", got)
			},
		},
		{
			name: "factory constructor; fill nil default",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryPtrConf, func() *ptestConfig { return nil })
				got := testNewOk(t, r, testNew, ptestFillConf)
				assert.Equal(t, ptestFilledValue, got)
			},
		},
		{
			name: "factory constructor; more than one fill conf panics",
			assert: func(t *testing.T, r *Registry, testNew New) {
				r.ptestRegister(ptestNewFactoryPtrConf)
				defer recoverExpectationFail(t)
				testNew(r, ptestFillConf, ptestFillConf)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewRegistry()
			testNew := (*Registry).ptestNew
			tt.assert(t, r, testNew)

			r = NewRegistry()
			testNew = (*Registry).ptestNewFactory
			tt.assert(t, r, testNew)
		})
	}
}

func TestDecode(t *testing.T) {
	r := NewRegistry()
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

	r.Register(ptestType(), "my-plugin", ptestNewConf, ptestDefaultConf)
	input := map[string]interface{}{
		"plugin": map[string]interface{}{
			nameKey: "my-plugin",
			"value": ptestFilledValue,
		},
	}
	type Config struct {
		Plugin ptestPlugin
	}
	var conf Config
	err := decode(input, &conf)
	assert.NoError(t, err)
	actualValue := conf.Plugin.(*ptestImpl).Value
	assert.Equal(t, ptestFilledValue, actualValue)
}

func recoverExpectationFail(t *testing.T) {
	r := recover()
	assert.NotNil(t, r)
	assert.Contains(t, r, "expectation failed")
}
