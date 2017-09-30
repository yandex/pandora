// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"reflect"

	"github.com/pkg/errors"
)

func NewRegistry() *Registry {
	return &Registry{make(map[reflect.Type]nameRegistry)}
}

type Registry struct {
	typeToNameReg map[reflect.Type]nameRegistry
}

func newNameRegistry() nameRegistry { return make(nameRegistry) }

type nameRegistry map[string]nameRegistryEntry

type nameRegistryEntry struct {
	constructor   constructor
	defaultConfig defaultConfigContainer
}

func (r *Registry) Register(
	pluginType reflect.Type, // plugin interface type
	name string,
	newPluginImpl interface{},
	newDefaultConfigOptional ...interface{},
) {
	expect(pluginType.Kind() == reflect.Interface, "plugin type should be interface, but have: %T", pluginType)
	expect(name != "", "empty name")
	nameReg := r.typeToNameReg[pluginType]
	if nameReg == nil {
		nameReg = newNameRegistry()
		r.typeToNameReg[pluginType] = nameReg
	}
	_, ok := nameReg[name]
	expect(!ok, "plugin %s with name %q had been already registered", pluginType, name)
	newDefaultConfig := getNewDefaultConfig(newDefaultConfigOptional)
	nameReg[name] = newNameRegistryEntry(pluginType, newPluginImpl, newDefaultConfig)
}

func (r *Registry) Lookup(pluginType reflect.Type) bool {
	_, ok := r.typeToNameReg[pluginType]
	return ok
}

func (r *Registry) LookupFactory(factoryType reflect.Type) bool {
	return isFactoryType(factoryType) && r.Lookup(factoryType.Out(0))
}

func (r *Registry) New(pluginType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	expect(pluginType.Kind() == reflect.Interface, "plugin type should be interface, but have: %T", pluginType)
	expect(name != "", "empty name")
	fillConf := getFillConf(fillConfOptional)
	registered, err := r.get(pluginType, name)
	if err != nil {
		return
	}
	conf, err := registered.defaultConfig.Get(fillConf)
	if err != nil {
		return nil, err
	}
	return registered.constructor.NewPlugin(conf)
}

func (r *Registry) NewFactory(factoryType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (factory interface{}, err error) {
	expect(isFactoryType(factoryType), "plugin factory type should be like `func() (PluginInterface, error)`, but have: %T", factoryType)
	expect(name != "", "empty name")
	fillConf := getFillConf(fillConfOptional)
	pluginType := factoryType.Out(0)
	registered, err := r.get(pluginType, name)
	if err != nil {
		return
	}
	var getMaybeConfig func() ([]reflect.Value, error)
	if registered.defaultConfig.configRequired() {
		getMaybeConfig = func() ([]reflect.Value, error) {
			return registered.defaultConfig.Get(fillConf)
		}
	} else if fillConf != nil {
		// Just check, that fillConf not fails, when there is no config fields.
		err := fillConf(&struct{}{})
		if err != nil {
			return nil, err
		}
	}
	return registered.constructor.NewFactory(factoryType, getMaybeConfig)
}

func newNameRegistryEntry(pluginType reflect.Type, newPluginImpl interface{}, newDefaultConfig interface{}) nameRegistryEntry {
	constructor := newPluginConstructor(pluginType, newPluginImpl)
	defaultConfig := newDefaultConfigContainer(reflect.TypeOf(newPluginImpl), newDefaultConfig)
	return nameRegistryEntry{constructor, defaultConfig}
}

func (r *Registry) get(pluginType reflect.Type, name string) (factory nameRegistryEntry, err error) {
	pluginReg, ok := r.typeToNameReg[pluginType]
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

// defaultConfigContainer contains default config creation logic.
// Zero value is valid and means that no config is needed.
type defaultConfigContainer struct {
	// !IsValid() if constructor accepts no arguments.
	// Otherwise type is func() <configType>.
	newValue reflect.Value
}

func newDefaultConfigContainer(constructorType reflect.Type, newDefaultConfig interface{}) defaultConfigContainer {
	if constructorType.NumIn() == 0 {
		expect(newDefaultConfig == nil, "constructor accept no config, but newDefaultConfig passed")
		return defaultConfigContainer{}
	}
	expect(constructorType.NumIn() == 1, "constructor should accept zero or one argument")
	configType := constructorType.In(0)
	expect(configType.Kind() == reflect.Struct ||
		configType.Kind() == reflect.Ptr && configType.Elem().Kind() == reflect.Struct,
		"unexpected config kind: %s; should be struct or struct pointer")
	newDefaultConfigType := reflect.FuncOf(nil, []reflect.Type{configType}, false)
	if newDefaultConfig == nil {
		value := reflect.MakeFunc(newDefaultConfigType,
			func(_ []reflect.Value) (results []reflect.Value) {
				// OPTIMIZE: create addressable.
				return []reflect.Value{reflect.Zero(configType)}
			})
		return defaultConfigContainer{value}
	}
	value := reflect.ValueOf(newDefaultConfig)
	expect(value.Type() == newDefaultConfigType,
		"newDefaultConfig should be func that accepts nothing, and returns constructor argument, but have type %T", newDefaultConfig)
	return defaultConfigContainer{value}
}

// In reflect pkg []Value used to call functions. It's easier to return it, that convert from pointer when needed.
func (e defaultConfigContainer) Get(fillConf func(fillAddr interface{}) error) (maybeConf []reflect.Value, err error) {
	var fillAddr interface{}
	if e.configRequired() {
		maybeConf, fillAddr = e.new()
	} else {
		var emptyStruct struct{}
		fillAddr = &emptyStruct // No fields to fill.
	}
	if fillConf != nil {
		err = fillConf(fillAddr)
		if err != nil {
			return nil, err
		}
	}
	return
}

func (e defaultConfigContainer) new() (maybeConf []reflect.Value, fillAddr interface{}) {
	if !e.configRequired() {
		panic("try to create config when not required")
	}
	conf := e.newValue.Call(nil)[0]
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

func (e defaultConfigContainer) configRequired() bool {
	return e.newValue.IsValid()
}

func getFillConf(fillConfOptional []func(conf interface{}) error) func(interface{}) error {
	expect(len(fillConfOptional) <= 1, "only fill config parameter could be passed")
	if len(fillConfOptional) == 0 {
		return nil
	}
	return fillConfOptional[0]
}

func getNewDefaultConfig(newDefaultConfigOptional []interface{}) interface{} {
	expect(len(newDefaultConfigOptional) <= 1, "too many arguments passed")
	if len(newDefaultConfigOptional) == 0 {
		return nil
	}
	return newDefaultConfigOptional[0]
}
