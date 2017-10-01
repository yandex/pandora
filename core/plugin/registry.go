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
	constructor   implConstructor
	defaultConfig defaultConfigContainer
}

// Register registers plugin constructor and optional default config factory,
// for given plugin interface type and plugin name.
// See package doc for type expectations details.
// Register designed to be called in package init func, so it panics if something go wrong.
// Panics if type expectations are violated.
// Panics if some constructor have been already registered for this (pluginType, name) pair.
// Register is thread unsafe.
//
// If constructor receive config argument, default config factory can be
// registered. Default config factory type should be: is func() <configType>.
// Default config factory is optional. If no default config factory has been
// registered, than plugin factory will receive zero config (zero struct or
// pointer to zero struct).
// Registered constructor will never receive nil config, even there
// are no registered default config factory, or default config is nil. Config
// will be pointer to zero config in such case.
func (r *Registry) Register(
	pluginType reflect.Type,
	name string,
	constructor interface{},
	newDefaultConfigOptional ...interface{}, // default config factory, or nothing.
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
	nameReg[name] = newNameRegistryEntry(pluginType, constructor, newDefaultConfig)
}

// Lookup returns true if any plugin constructor has been registered for given
// type.
func (r *Registry) Lookup(pluginType reflect.Type) bool {
	_, ok := r.typeToNameReg[pluginType]
	return ok
}

// LookupFactory returns true if factoryType looks like func() (SomeInterface[, error])
// and any plugin constructor has been registered for SomeInterface.
// That is, you may create instance of this factoryType using this registry.
func (r *Registry) LookupFactory(factoryType reflect.Type) bool {
	return isFactoryType(factoryType) && r.Lookup(factoryType.Out(0))
}

// New creates plugin using registered plugin constructor. Returns error if creation
// failed or no plugin were registered for given type and name.
// Passed fillConf called on created config before calling plugin factory.
// fillConf argument is always valid struct pointer, even if plugin factory
// receives no config: fillConf is called on empty struct pointer in such case.
// fillConf error fails plugin creation.
// New is thread safe, if there is no concurrent Register calls.
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

// NewFactory behaves like New, but creates factory func() (PluginInterface[, error]), that on call
// creates New plugin by registered factory.
// If registered constructor is <newPlugin> config is created filled for every factory call,
// if <newFactory, that only once for factory creation.
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

func newNameRegistryEntry(pluginType reflect.Type, constructor interface{}, newDefaultConfig interface{}) nameRegistryEntry {
	implConstructor := newImplConstructor(pluginType, constructor)
	defaultConfig := newDefaultConfigContainer(reflect.TypeOf(constructor), newDefaultConfig)
	return nameRegistryEntry{implConstructor, defaultConfig}
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
