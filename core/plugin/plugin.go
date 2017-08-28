// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"reflect"
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

// LookupFactory returns true if factoryType looks like func() (SomeInterface[, error])
// and any plugin factory has been registered for SomeInterface.
func LookupFactory(factoryType reflect.Type) bool {
	return defaultRegistry.LookupFactory(factoryType)
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

// FactoryPluginType returns (SomeInterface, true) if factoryType looks like func() (SomeInterface[, error])
// or (nil, false) otherwise.
func FactoryPluginType(factoryType reflect.Type) (plugin reflect.Type, ok bool) {
	if isFactoryType(factoryType) {
		return factoryType.Out(0), true
	}
	return
}

// isFactoryType returns true, if type looks like func() (SomeInterface[, error])
func isFactoryType(t reflect.Type) bool {
	hasProperParamsNum := t.Kind() == reflect.Func &&
		t.NumIn() == 0 &&
		(t.NumOut() == 1 || t.NumOut() == 2)
	if !hasProperParamsNum {
		return false
	}
	if t.Out(0).Kind() != reflect.Interface {
		return false
	}
	if t.NumOut() == 1 {
		return true
	}
	return t.Out(1) == errorType
}

var defaultRegistry = newTypeRegistry()
