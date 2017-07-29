// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

// Package plugin provides a generic inversion of control model for making
// extensible Go packages, libraries, and applications. Like
// github.com/progrium/go-extpoints, but reflect based: doesn't require code
// generation, but have more overhead; provide more flexibility, but less type
// safety. It allows to register factory for some plugin interface, and create
// new plugin instances by registered factory.
// Main feature is flexible plugin configuration: plugin factory can
// accept config struct, that could be filled by passed hook. Config default
// values could be provided by registering default config factory.
// Such flexibility can be used to decode structured text (json/yaml/etc) into
// struct with plugin interface fields.
//
// Type expectations.
// Plugin factory type should be:
// func([config <configType>]) (<pluginImpl> [, error])
// where configType kind is struct or struct pointer, and pluginImpl implements
// plugin interface. Plugin factory will never receive nil config, even there
// are no registered default config factory, or default config is nil. Config
// will be pointer to zero config in such case.
// If plugin factory receive config argument, default config factory can be
// registered. Default config factory type should be: is func() <configType>.
// Default config factory is optional. If no default config factory has been
// registered, than plugin factory will receive zero config (zero struct or
// pointer to zero struct).
//
// Note, that plugin interface type could be taken as reflect.TypeOf((*PluginInterface)(nil)).Elem().
package plugin

import (
	"fmt"
	"reflect"

	"github.com/facebookgo/stackerr"
)

// Register registers plugin factory and optional default config factory,
// for given plugin interface type and plugin name.
// See package doc for type expectations details.
// Register designed to be called in package init func, so it panics if type
// expectations were failed. Register is thread unsafe.
func Register(
	pluginType reflect.Type,
	name string,
	newPluginImpl interface{},
	newDefaultConfigOptional ...interface{},
) {
	defaultRegistry.Register(pluginType, name, newPluginImpl, newDefaultConfigOptional...)
}

// Lookup returns true if any plugin factory has been registered for given
// type.
func Lookup(pluginType reflect.Type) bool {
	return defaultRegistry.Lookup(pluginType)
}

func LookupFactory(pluginType reflect.Type) bool {
	return defaultRegistry.LookupFactory(pluginType)
}

// New creates plugin by registered plugin factory. Returns error if creation
// failed or no plugin were registered for given type and name.
// Passed fillConf called on created config before calling plugin factory.
// fillConf argument is always valid struct pointer, even if plugin factory
// receives no config: fillConf is called on empty struct pointer in such case.
// fillConf error fails plugin creation.
// New is thread safe, if there is no concurrent Register calls.
func New(pluginType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	return defaultRegistry.New(pluginType, name, fillConfOptional...)
}

// TODO (skipor): add support for `func() PluginInterface` factories, that
// panics on error.
// TODO (skipor): add NewSharedConfigsFactory that decodes config once and use
// it to create all plugin instances.

// NewFactory behaves like New, but creates factory func() (PluginInterface, error), that on call
// creates New plugin by registered factory.
// New config is created filled for every factory call.
func NewFactory(factoryType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (factory interface{}, err error) {
	return defaultRegistry.NewFactory(factoryType, name, fillConfOptional...)
}

// PtrType is helper to extract plugin types.
// Example: plugin.PtrType((*PluginInterface)(nil)) instead of
// reflect.TypeOf((*PluginInterface)(nil)).Elem()
func PtrType(ptr interface{}) reflect.Type {
	t := reflect.TypeOf(ptr)
	if t.Kind() != reflect.Ptr {
		panic("passed value is not pointer")
	}
	return t.Elem()
}

func IsFactoryType(t reflect.Type) bool {
	return t.Kind() == reflect.Func &&
		t.NumIn() == 0 &&
		t.NumOut() == 2 &&
		t.Out(0).Kind() == reflect.Interface &&
		t.Out(1) == errorType
}

func FactoryPluginType(factory reflect.Type) (plugin reflect.Type, ok bool) {
	if IsFactoryType(factory) {
		return factory.Out(0), true
	}
	return
}

type nameRegistryEntry struct {
	// newPluginImpl type is func([config <configType>]) (<pluginImpl> [, error]),
	// where configType kind is struct or struct pointer.
	newPluginImpl reflect.Value
	// newDefaultConfig type is func() <configType>. Zero if newPluginImpl accepts no arguments.
	newDefaultConfig reflect.Value
}

type nameRegistry map[string]nameRegistryEntry

func newNameRegistry() nameRegistry { return make(nameRegistry) }

type typeRegistry map[reflect.Type]nameRegistry

var defaultRegistry = newTypeRegistry()

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
	return IsFactoryType(factoryType) && r.Lookup(factoryType.Out(0))
}

func (r typeRegistry) New(pluginType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	expect(pluginType.Kind() == reflect.Interface, "plugin type should be interface, but have: %T", pluginType)
	expect(name != "", "empty name")
	fillConf := getFillConf(fillConfOptional)
	registered, err := r.get(pluginType, name)
	if err != nil {
		return
	}
	confOptional, fillAddr := registered.NewDefaultConfig()
	if fillConf != nil {
		err = fillConf(fillAddr)
		if err != nil {
			return
		}
	}
	return registered.NewPlugin(confOptional)
}

func (r typeRegistry) NewFactory(factoryType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (factory interface{}, err error) {
	expect(IsFactoryType(factoryType), "plugin factory type should be like `func() (PluginInterface, error)`, but have: %T", factoryType)
	expect(name != "", "empty name")
	fillConf := getFillConf(fillConfOptional)
	pluginType := factoryType.Out(0)
	registered, err := r.get(pluginType, name)
	if err != nil {
		return
	}
	factory = reflect.MakeFunc(factoryType, func(in []reflect.Value) (out []reflect.Value) {
		conf, fillAddr := registered.NewDefaultConfig()
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

func (e nameRegistryEntry) NewDefaultConfig() (confOptional []reflect.Value, fillAddr interface{}) {
	if e.newPluginImpl.Type().NumIn() == 0 {
		var emptyStruct struct{}
		fillAddr = &emptyStruct // No fields to fill.
		return
	}
	conf := e.newDefaultConfig.Call(nil)[0]
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
	confOptional = []reflect.Value{conf}
	return
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
		newPluginImpl:    reflect.ValueOf(newPluginImpl),
		newDefaultConfig: reflect.ValueOf(newDefaultConfig),
	}
}

func (r typeRegistry) get(pluginType reflect.Type, name string) (factory nameRegistryEntry, err error) {
	pluginReg, ok := r[pluginType]
	if !ok {
		err = stackerr.Newf("no plugins for type %s has been registered", pluginType)
		return
	}
	factory, ok = pluginReg[name]
	if !ok {
		err = stackerr.Newf("no plugins of type %s has been registered for name %s", pluginType, name)
	}
	return
}

func expect(b bool, msg string, args ...interface{}) {
	if !b {
		panic(fmt.Sprintf("expectation failed: "+msg, args...))
	}
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()
