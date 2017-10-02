// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"fmt"
	"reflect"
)

// DefaultRegistry returns default Registry used for package Registry like functions.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Register is DefaultRegistry().Register shortcut.
func Register(
	pluginType reflect.Type,
	name string,
	newPluginImpl interface{},
	newDefaultConfigOptional ...interface{},
) {
	DefaultRegistry().Register(pluginType, name, newPluginImpl, newDefaultConfigOptional...)
}

// Lookup is DefaultRegistry().Lookup shortcut.
func Lookup(pluginType reflect.Type) bool {
	return DefaultRegistry().Lookup(pluginType)
}

// LookupFactory is DefaultRegistry().LookupFactory shortcut.
func LookupFactory(factoryType reflect.Type) bool {
	return DefaultRegistry().LookupFactory(factoryType)
}

// New is DefaultRegistry().New shortcut.
func New(pluginType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	return defaultRegistry.New(pluginType, name, fillConfOptional...)
}

// NewFactory is DefaultRegistry().NewFactory shortcut.
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

var defaultRegistry = NewRegistry()

var errorType = reflect.TypeOf((*error)(nil)).Elem()

func expect(b bool, msg string, args ...interface{}) {
	if !b {
		panic(fmt.Sprintf("expectation failed: "+msg, args...))
	}
}
