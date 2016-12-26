// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"fmt"
	"reflect"

	"github.com/facebookgo/stackerr"
)

type pluginFactory struct {
	// newPluginImpl type is func([config <configType>]) (<pluginImpl> [, error]),
	// where configType kind is struct, struct pointer or map.
	newPluginImpl reflect.Value
	// newDefaultConfig type is func() <configType>. Zero if newPluginImpl accepts no arguments.
	newDefaultConfig reflect.Value
}

type pluginRegistry map[string]pluginFactory

func newPluginRegistry() pluginRegistry { return make(pluginRegistry) }

type registry map[reflect.Type]pluginRegistry

var defaultRegistry = newRegistry()

func newRegistry() registry { return make(registry) }

func (r registry) Lookup(pluginType reflect.Type) bool {
	_, ok := r[pluginType]
	return ok
}

func (r registry) Register(
	pluginType reflect.Type, // plugin interface type
	name string,
	newPluginImpl interface{},
	newDefaultConfigOptional ...interface{},
) {
	basicCheck(pluginType, name)
	pluginReg := r[pluginType]
	if pluginReg == nil {
		pluginReg = newPluginRegistry()
		r[pluginType] = pluginReg
	}
	_, ok := pluginReg[name]
	expect(!ok, "plugin %s with name %q is already registered", pluginType, name)
	pluginReg[name] = newPluginFactory(pluginType, newPluginImpl, newDefaultConfigOptional...)
}

func newPluginFactory(pluginType reflect.Type, newPluginImpl interface{}, newDefaultConfigOptional ...interface{}) pluginFactory {
	newPluginImplType := reflect.TypeOf(newPluginImpl)
	expect(newPluginImplType.Kind() == reflect.Func, "newPluginImpl should be func")
	expect(newPluginImplType.NumIn() <= 1, "newPluginImple should accept config or nothing")
	expect(1 <= newPluginImplType.NumOut() && newPluginImplType.NumOut() <= 2,
		"newPluginImple should return plugin implementation, and optionally error")
	pluginImplType := newPluginImplType.Out(0)
	expect(pluginImplType.Implements(pluginType), "pluginImpl should implement plugin interface")
	if newPluginImplType.NumOut() == 2 {
		errorType := reflect.TypeOf((*error)(nil)).Elem()
		expect(newPluginImplType.Out(1) == errorType, "pluginImpl should have no second return value, or it should be error")
	}

	if newPluginImplType.NumIn() == 0 {
		expect(len(newDefaultConfigOptional) == 0, "newPluginImpl accept no config, but newDefaultConfig passed")
		return pluginFactory{
			newPluginImpl: reflect.ValueOf(newPluginImpl),
		}
	}

	expect(len(newDefaultConfigOptional) <= 1, "only one default config newPluginImpl could be passed")
	configType := newPluginImplType.In(0)
	switch configType.Kind() {
	case reflect.Struct:
	case reflect.Ptr:
		// TODO: remove map configs?
	//case reflect.Map:
	//	expect(configType.Key() == reflect.TypeOf(""), "config map key should be string, but is: %s", configType.Key())
	default:
		expect(false, "unexpected config kind: %s; should be struct, struct pointer or map")
	}

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
	return pluginFactory{
		newPluginImpl:    reflect.ValueOf(newPluginImpl),
		newDefaultConfig: reflect.ValueOf(newDefaultConfig),
	}
}

func (r registry) get(pluginType reflect.Type, name string) (factory pluginFactory, err error) {
	pluginReg, ok := r[pluginType]
	if !ok {
		err = stackerr.Newf("no plugins for type %s has been registered", pluginType)
		return
	}
	factory, ok = pluginReg[name]
	if !ok {
		err = stackerr.Newf("no plugins  of type %s has been registered for name %s", pluginType, name)
	}
	return
}

func (r registry) New(pluginType reflect.Type, name string, fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	basicCheck(pluginType, name)
	expect(len(fillConfOptional) <= 1, "only fill config parameter could be passed")
	var fillConf func(interface{}) error
	if len(fillConfOptional) != 0 {
		fillConf = fillConfOptional[0]
	}

	var factory pluginFactory
	factory, err = r.get(pluginType, name)
	if err != nil {
		return
	}
	var args []reflect.Value
	var conf interface{}
	if factory.newPluginImpl.Type().NumIn() == 0 {
		var emptyStruct struct{}
		conf = &emptyStruct // Check than fill conf expects nothing.
	} else {
		args = append(factory.newDefaultConfig.Call(nil))
		switch args[0].Kind() {
		case reflect.Struct:
			// Config can be filled only by pointer.
			if !args[0].CanAddr() {
				// Can't pass pointer into decoder. Let's make new addressable!
				newArg := reflect.New(args[0].Type()).Elem()
				newArg.Set(args[0])
				args[0] = newArg
			}
			conf = args[0].Addr().Interface()
		case reflect.Ptr:
			if args[0].IsNil() {
				// Can't fill nil config. Init with zero.
				args[0] = reflect.New(args[0].Type().Elem())
			}
			conf = args[0].Interface()
		default:
			panic("unexpected type " + args[0].String())
		}
	}
	if fillConf != nil {
		err = fillConf(conf)
		if err != nil {
			return
		}
	}
	out := factory.newPluginImpl.Call(args)
	plugin = out[0].Interface()
	if len(out) > 1 {
		err = out[1].Interface().(error)
	}
	return
}

func basicCheck(pluginType reflect.Type, name string) {
	expect(pluginType.Kind() == reflect.Interface, "plugin type should be interface, but have: %T", pluginType)
	expect(name != "", "empty name")
}

func expect(b bool, msg string, args ...interface{}) {
	if !b {
		panic(fmt.Sprintf("expectation failed: "+msg, args...))
	}
}
