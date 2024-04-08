package plugin

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_PluginConstructor_ExpectationsFailed(t *testing.T) {
	tests := []struct {
		name      string
		newPlugin any
	}{
		{"not func", errors.New("that is not constructor")},
		{"not implements", func() struct{} { panic("") }},
		{"too many args", func(_, _ ptestConfig) ptestPlugin { panic("") }},
		{"too many return valued", func() (_ ptestPlugin, _, _ error) { panic("") }},
		{"second return value is not error", func() (_, _ ptestPlugin) { panic("") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer recoverExpectationFail(t)
			newPluginConstructor(ptestType(), tt.newPlugin)
		})
	}
}

func Test_PluginConstructor_NewPlugin(t *testing.T) {
	newPlugin := func(newPlugin interface{}, maybeConf []reflect.Value) (interface{}, error) {
		testee := newPluginConstructor(ptestType(), newPlugin)
		return testee.NewPlugin(maybeConf)
	}

	t.Run("", func(t *testing.T) {
		plugin, err := newPlugin(ptestNew, nil)
		assert.NoError(t, err)
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("more that plugin", func(t *testing.T) {
		plugin, err := newPlugin(ptestNewMoreThan, nil)
		assert.NoError(t, err)
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("config", func(t *testing.T) {
		plugin, err := newPlugin(ptestNewConf, confToMaybe(ptestDefaultConf()))
		assert.NoError(t, err)
		ptestExpectConfigValue(t, plugin, ptestDefaultValue)
	})

	t.Run("failed", func(t *testing.T) {
		plugin, err := newPlugin(ptestNewErrFailing, nil)
		assert.ErrorIs(t, err, ptestCreateFailedErr)
		assert.Nil(t, plugin)
	})

}

func Test_PluginConstructor_NewFactory(t *testing.T) {
	newFactoryOK := func(newPlugin interface{}, factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) interface{} {
		testee := newPluginConstructor(ptestType(), newPlugin)
		factory, err := testee.NewFactory(factoryType, getMaybeConf)
		require.NoError(t, err)
		return factory
	}

	t.Run("same type - no wrap", func(t *testing.T) {
		factory := newFactoryOK(ptestNew, ptestNewType(), nil)
		expectSameFunc(t, factory, ptestNew)
	})

	t.Run(" new impl", func(t *testing.T) {
		factory := newFactoryOK(ptestNewImpl, ptestNewType(), nil)
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		plugin := f()
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("more than", func(t *testing.T) {
		factory := newFactoryOK(ptestNewMoreThan, ptestNewType(), nil)
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		plugin := f()
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("add err", func(t *testing.T) {
		factory := newFactoryOK(ptestNew, ptestNewErrType(), nil)
		f, ok := factory.(func() (ptestPlugin, error))
		assert.True(t, ok)
		plugin, err := f()
		assert.NoError(t, err)
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("trim nil err", func(t *testing.T) {
		factory := newFactoryOK(ptestNewErr, ptestNewType(), nil)
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		plugin := f()
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("config", func(t *testing.T) {
		factory := newFactoryOK(ptestNewConf, ptestNewType(), confToGetMaybe(ptestDefaultConf()))
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		plugin := f()
		ptestExpectConfigValue(t, plugin, ptestDefaultValue)
	})

	t.Run("new factory, get config failed", func(t *testing.T) {
		factory := newFactoryOK(ptestNewConf, ptestNewErrType(), errToGetMaybe(ptestConfigurationFailedErr))
		f, ok := factory.(func() (ptestPlugin, error))
		assert.True(t, ok)
		plugin, err := f()
		assert.ErrorIs(t, err, ptestConfigurationFailedErr)
		assert.Nil(t, plugin)
	})

	t.Run("no err, get config failed, throw panic", func(t *testing.T) {
		factory := newFactoryOK(ptestNewConf, ptestNewType(), errToGetMaybe(ptestConfigurationFailedErr))
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		func() {
			defer func() {
				r := recover()
				assert.Equal(t, ptestConfigurationFailedErr, r)
			}()
			f()
		}()
	})

	t.Run("panic on trim non nil err", func(t *testing.T) {
		factory := newFactoryOK(ptestNewErrFailing, ptestNewType(), nil)
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		func() {
			defer func() {
				r := recover()
				assert.Equal(t, ptestCreateFailedErr, r)
			}()
			f()
		}()
	})

}

func TestFactoryConstructorExpectationsFailed(t *testing.T) {
	tests := []struct {
		name      string
		newPlugin any
	}{
		{"not func", errors.New("that is not constructor")},
		{"returned not func", func() error { panic("") }},
		{"too many args", func(_, _ ptestConfig) func() ptestPlugin { panic("") }},
		{"too many return valued", func() (func() ptestPlugin, error, error) { panic("") }},
		{"second return value is not error", func() (func() ptestPlugin, ptestPlugin) { panic("") }},
		{"factory accepts conf", func() func(config ptestConfig) ptestPlugin { panic("") }},
		{"not implements", func() func() struct{} { panic("") }},
		{"factory too many args", func() func(_, _ ptestConfig) ptestPlugin { panic("") }},
		{"factory too many return valued", func() func() (_ ptestPlugin, _, _ error) { panic("") }},
		{"factory second return value is not error", func() func() (_, _ ptestPlugin) { panic("") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer recoverExpectationFail(t)
			newFactoryConstructor(ptestType(), tt.newPlugin)
		})
	}
}

func TestFactoryConstructorNewPlugin(t *testing.T) {
	newPlugin := func(newFactory interface{}, maybeConf []reflect.Value) (interface{}, error) {
		testee := newFactoryConstructor(ptestType(), newFactory)
		return testee.NewPlugin(maybeConf)
	}

	t.Run("", func(t *testing.T) {
		plugin, err := newPlugin(ptestNewFactory, nil)
		assert.NoError(t, err)
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("impl", func(t *testing.T) {
		plugin, err := newPlugin(ptestNewFactoryImpl, nil)
		assert.NoError(t, err)
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("impl more than", func(t *testing.T) {
		plugin, err := newPlugin(ptestNewFactoryMoreThan, nil)
		assert.NoError(t, err)
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("config", func(t *testing.T) {
		plugin, err := newPlugin(ptestNewFactoryConf, confToMaybe(ptestDefaultConf()))
		assert.NoError(t, err)
		ptestExpectConfigValue(t, plugin, ptestDefaultValue)
	})

	t.Run("failed", func(t *testing.T) {
		plugin, err := newPlugin(ptestNewFactoryErrFailing, nil)
		assert.ErrorIs(t, err, ptestCreateFailedErr)
		assert.Nil(t, plugin)
	})

	t.Run("factory failed", func(t *testing.T) {
		plugin, err := newPlugin(ptestNewFactoryFactoryErrFailing, nil)
		assert.ErrorIs(t, err, ptestCreateFailedErr)
		assert.Nil(t, plugin)
	})
}

func TestFactoryConstructorNewFactory(t *testing.T) {
	newFactory := func(newFactory interface{}, factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) (interface{}, error) {
		testee := newFactoryConstructor(ptestType(), newFactory)
		return testee.NewFactory(factoryType, getMaybeConf)
	}
	newFactoryOK := func(newF interface{}, factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) interface{} {
		factory, err := newFactory(newF, factoryType, getMaybeConf)
		require.NoError(t, err)
		return factory
	}

	t.Run("no err, same type - no wrap", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactory, ptestNewType(), nil)
		expectSameFunc(t, factory, ptestNew)
	})

	t.Run("has err, same type - no wrap", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactoryFactoryErr, ptestNewErrType(), nil)
		expectSameFunc(t, factory, ptestNewErr)
	})

	t.Run("from new impl", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactoryImpl, ptestNewType(), nil)
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		plugin := f()
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("from new impl", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactoryMoreThan, ptestNewType(), nil)
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		plugin := f()
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("add err", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactory, ptestNewErrType(), nil)
		f, ok := factory.(func() (ptestPlugin, error))
		assert.True(t, ok)
		plugin, err := f()
		assert.NoError(t, err)
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("factory construction not failed", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactoryErr, ptestNewType(), nil)
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		plugin := f()
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("trim nil err", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactoryFactoryErr, ptestNewType(), nil)
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		plugin := f()
		ptestExpectConfigValue(t, plugin, ptestInitValue)
	})

	t.Run("config", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactoryConf, ptestNewType(), confToGetMaybe(ptestDefaultConf()))
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		plugin := f()
		ptestExpectConfigValue(t, plugin, ptestDefaultValue)
	})

	t.Run("get config failed", func(t *testing.T) {
		factory, err := newFactory(ptestNewFactoryConf, ptestNewErrType(), errToGetMaybe(ptestConfigurationFailedErr))
		assert.Error(t, err, ptestConfigurationFailedErr)
		assert.Nil(t, factory)
	})

	t.Run("factory create failed", func(t *testing.T) {
		factory, err := newFactory(ptestNewFactoryErrFailing, ptestNewErrType(), nil)
		assert.ErrorIs(t, err, ptestCreateFailedErr)
		assert.Nil(t, factory)
	})

	t.Run("plugin create failed", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactoryFactoryErrFailing, ptestNewErrType(), nil)
		f, ok := factory.(func() (ptestPlugin, error))
		assert.True(t, ok)
		plugin, err := f()
		assert.ErrorIs(t, err, ptestCreateFailedErr)
		assert.Nil(t, plugin)
	})

	t.Run("panic on trim non nil err", func(t *testing.T) {
		factory := newFactoryOK(ptestNewFactoryFactoryErrFailing, ptestNewType(), nil)
		f, ok := factory.(func() ptestPlugin)
		assert.True(t, ok)
		func() {
			defer func() {
				r := recover()
				assert.Equal(t, ptestCreateFailedErr, r)
			}()
			f()
		}()
	})

}

func confToMaybe(conf interface{}) []reflect.Value {
	if conf != nil {
		return []reflect.Value{reflect.ValueOf(conf)}
	}
	return nil
}

func confToGetMaybe(conf interface{}) func() ([]reflect.Value, error) {
	return func() ([]reflect.Value, error) {
		return confToMaybe(conf), nil
	}
}

func errToGetMaybe(err error) func() ([]reflect.Value, error) {
	return func() ([]reflect.Value, error) {
		return nil, err
	}
}

func expectSameFunc(t *testing.T, f1, f2 interface{}) {
	s1 := fmt.Sprint(f1)
	s2 := fmt.Sprint(f2)
	assert.Equal(t, s1, s2)
}
