// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"fmt"
	"reflect"

	"github.com/pkg/errors"
)

type newDefaultConfigContainer struct {
	// value type is func() <configType>. !IsValid() if newPluginImpl accepts no arguments.
	value reflect.Value
}

// In reflect pkg []Value used to call functions. It's easier to return it, that convert from pointer when needed.
func (e newDefaultConfigContainer) Get() (maybeConf []reflect.Value, fillAddr interface{}) {
	configRequired := e.value.IsValid()
	if !configRequired {
		var emptyStruct struct{}
		fillAddr = &emptyStruct // No fields to fill.
		return
	}
	conf := e.value.Call(nil)[0]
	switch conf.Kind() {
	case reflect.Struct:
		// Config can be filled only by pointer.
		if !conf.CanAddr() {
			// Can't address to pass pointer into decoder. Let's make New addressable!
			newArg := reflect.New(conf.Type()).Elem()
			newArg.Set(conf)
			conf = newArg
		}
		fillAddr = conf.Addr().Interface()
	case reflect.Ptr:
		if conf.IsNil() {
			// Can't fill nil config. Init with zero.
			conf = reflect.New(conf.Type().Elem())
		}
		fillAddr = conf.Interface()
	default:
		panic("unexpected type " + conf.String())
	}
	maybeConf = []reflect.Value{conf}
	return
}

type nameRegistryEntry struct {
	// newPluginImpl type is func([config <configType>]) (<pluginImpl> [, error]),
	// where configType kind is struct or struct pointer.
	newPluginImpl reflect.Value
	defaultConfig newDefaultConfigContainer
}

type nameRegistry map[string]nameRegistryEntry

func newNameRegistry() nameRegistry { return make(nameRegistry) }

type typeRegistry map[reflect.Type]nameRegistry

func newTypeRegistry() typeRegistry { return make(typeRegistry) }

func (r typeRegistry) Register(
	pluginType reflect.Type, // plugin interface type
	name string,
	newPluginImpl interface{},
	newDefaultConfigOptional ...interface{},
) {
	expect(pluginType.Kind() == reflect.Interface, "plugin type should be interface, but have: %T", pluginType)
	expect(name != "", "empty name")
	pluginReg := r[pluginType]
	if pluginReg == nil {
		pluginReg = newNameRegistry()
		r[pluginType] = pluginReg
	}
	_, ok := pluginReg[name]
	expect(!ok, "plugin %s with name %q had been already registered", pluginType, name)
	pluginReg[name] = newNameRegistryEntry(pluginType, newPluginImpl, newDefaultConfigOptional...)
}

func (r typeRegistry) Lookup(pluginType reflect.Type) bool {
	_, ok := r[pluginType]
	return ok
}

func (r typeRegistry) LookupFactory(factoryType reflect.Type) bool {
	return isFactoryType(factoryType) && r.Lookup(factoryType.Out(0))
}

func (r typeRegistry) New(pluginType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	expect(pluginType.Kind() == reflect.Interface, "plugin type should be interface, but have: %T", pluginType)
	expect(name != "", "empty name")
	fillConf := getFillConf(fillConfOptional)
	registered, err := r.get(pluginType, name)
	if err != nil {
		return
	}
	conf, fillAddr := registered.defaultConfig.Get()
	if fillConf != nil {
		err = fillConf(fillAddr)
		if err != nil {
			return
		}
	}
	return registered.NewPlugin(conf)
}

func (r typeRegistry) NewFactory(factoryType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (factory interface{}, err error) {
	expect(isFactoryType(factoryType), "plugin factory type should be like `func() (PluginInterface, error)`, but have: %T", factoryType)
	expect(name != "", "empty name")
	fillConf := getFillConf(fillConfOptional)
	pluginType := factoryType.Out(0)
	registered, err := r.get(pluginType, name)
	if err != nil {
		return
	}
	factory = reflect.MakeFunc(factoryType, func(in []reflect.Value) (out []reflect.Value) {
		conf, fillAddr := registered.defaultConfig.Get()
		if fillConf != nil {
			// Check that config is correct.
			err := fillConf(fillAddr)
			if err != nil {
				return []reflect.Value{reflect.Zero(pluginType), reflect.ValueOf(&err).Elem()}
			}
		}
		out = registered.newPluginImpl.Call(conf)
		if out[0].Type() != pluginType {
			// Not plugin, but its implementation.
			impl := out[0]
			out[0] = reflect.New(pluginType).Elem()
			out[0].Set(impl)
		}

		if len(out) < 2 {
			// Registered newPluginImpl can return no error, but we should.
			out = append(out, reflect.Zero(errorType))
		}
		return
	}).Interface()
	return
}

func getFillConf(fillConfOptional []func(conf interface{}) error) func(interface{}) error {
	expect(len(fillConfOptional) <= 1, "only fill config parameter could be passed")
	if len(fillConfOptional) == 0 {
		return nil
	}
	return fillConfOptional[0]
}

func (e nameRegistryEntry) NewPlugin(confOptional []reflect.Value) (plugin interface{}, err error) {
	out := e.newPluginImpl.Call(confOptional)
	plugin = out[0].Interface()
	if len(out) > 1 {
		err, _ = out[1].Interface().(error)
	}
	return
}

func newNameRegistryEntry(pluginType reflect.Type, newPluginImpl interface{}, newDefaultConfigOptional ...interface{}) nameRegistryEntry {
	newPluginImplType := reflect.TypeOf(newPluginImpl)
	expect(newPluginImplType.Kind() == reflect.Func, "newPluginImpl should be func")
	expect(newPluginImplType.NumIn() <= 1, "newPluginImple should accept config or nothing")
	expect(1 <= newPluginImplType.NumOut() && newPluginImplType.NumOut() <= 2,
		"newPluginImple should return plugin implementation, and optionally error")
	pluginImplType := newPluginImplType.Out(0)
	expect(pluginImplType.Implements(pluginType), "pluginImpl should implement plugin interface")
	if newPluginImplType.NumOut() == 2 {
		expect(newPluginImplType.Out(1) == errorType, "pluginImpl should have no second return value, or it should be error")
	}

	if newPluginImplType.NumIn() == 0 {
		expect(len(newDefaultConfigOptional) == 0, "newPluginImpl accept no config, but newDefaultConfig passed")
		return nameRegistryEntry{
			newPluginImpl: reflect.ValueOf(newPluginImpl),
		}
	}

	expect(len(newDefaultConfigOptional) <= 1, "only one default config newPluginImpl could be passed")
	configType := newPluginImplType.In(0)
	expect(configType.Kind() == reflect.Struct ||
		configType.Kind() == reflect.Ptr && configType.Elem().Kind() == reflect.Struct,
		"unexpected config kind: %s; should be struct or struct pointer or map")

	newDefaultConfigType := reflect.FuncOf(nil, []reflect.Type{configType}, false)
	var newDefaultConfig interface{}
	if len(newDefaultConfigOptional) != 0 {
		newDefaultConfig = newDefaultConfigOptional[0]
		expect(reflect.TypeOf(newDefaultConfig) == newDefaultConfigType,
			"newDefaultConfig should be func that accepst nothing, and returns newPluiginImpl argument, but have type %T", newDefaultConfig)
	} else {
		newDefaultConfig = reflect.MakeFunc(newDefaultConfigType,
			func(_ []reflect.Value) (results []reflect.Value) {
				return []reflect.Value{reflect.Zero(configType)}
			}).Interface()
	}
	return nameRegistryEntry{
		newPluginImpl: reflect.ValueOf(newPluginImpl),
		defaultConfig: newDefaultConfigContainer{reflect.ValueOf(newDefaultConfig)},
	}
}

func (r typeRegistry) get(pluginType reflect.Type, name string) (factory nameRegistryEntry, err error) {
	pluginReg, ok := r[pluginType]
	if !ok {
		err = errors.Errorf("no plugins for type %s has been registered", pluginType)
		return
	}
	factory, ok = pluginReg[name]
	if !ok {
		err = errors.Errorf("no plugins of type %s has been registered for name %s", pluginType, name)
	}
	return
}

func expect(b bool, msg string, args ...interface{}) {
	if !b {
		panic(fmt.Sprintf("expectation failed: "+msg, args...))
	}
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()
