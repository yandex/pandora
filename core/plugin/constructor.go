// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"fmt"
	"reflect"
)

// constructor interface representing ability to create some plugin interface
// implementations and is't factory functions. Usually it wraps some newPlugin or newFactory function, and
// use is as implementation creator.
// constructor expects, that caller pass correct maybeConf value, that can be
// passed to underlying implementation creator.
type constructor interface {
	// NewPlugin constructs plugin implementation.
	NewPlugin(maybeConf []reflect.Value) (plugin interface{}, err error)
	// getMaybeConf may be nil, if no config required.
	// Underlying implementation creator may require new config for every instance create.
	// If so, then getMaybeConf will be called on every instance create. Otherwise, only once.
	NewFactory(factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) (pluginFactory interface{}, err error)
}

func newPluginConstructor(pluginType reflect.Type, newPlugin interface{}) *pluginConstructor {
	newPluginType := reflect.TypeOf(newPlugin)
	expect(newPluginType.Kind() == reflect.Func, "plugin constructor should be func")
	expect(newPluginType.NumIn() <= 1, "plugin constructor should accept config or nothing")
	expect(1 <= newPluginType.NumOut() && newPluginType.NumOut() <= 2,
		"plugin constructor should return plugin implementation, and optionally error")
	pluginImplType := newPluginType.Out(0)
	expect(pluginImplType.Implements(pluginType), "plugin constructor should implement plugin interface")
	if newPluginType.NumOut() == 2 {
		expect(newPluginType.Out(1) == errorType, "plugin constructor should have no second return value, or it should be error")
	}
	newPluginVal := reflect.ValueOf(newPlugin)
	return &pluginConstructor{pluginType, newPluginVal}
}

// pluginConstructor is abstract constructor of some pluginType interface implementations
// and it's factory functions, using some func newPlugin func.
type pluginConstructor struct {
	pluginType reflect.Type
	// newPlugin type is func([config <configType>]) (<pluginImpl> [, error]),
	// where configType kind is struct or struct pointer.
	newPlugin reflect.Value
}

func (c *pluginConstructor) NewPlugin(maybeConf []reflect.Value) (plugin interface{}, err error) {
	out := c.newPlugin.Call(maybeConf)
	plugin = out[0].Interface()
	if len(out) > 1 {
		err, _ = out[1].Interface().(error)
	}
	return
}

func (c *pluginConstructor) NewFactory(factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) (interface{}, error) {
	// FIXME: TEST no config and same type return newPlugin interface
	// FIXME: TEST no error returning factories
	if c.newPlugin.Type() == factoryType {
		return c.newPlugin.Interface(), nil
	}
	return reflect.MakeFunc(factoryType, func(in []reflect.Value) []reflect.Value {
		var maybeConf []reflect.Value
		if getMaybeConf != nil {
			var err error
			maybeConf, err = getMaybeConf()
			if err != nil {
				switch factoryType.NumOut() {
				case 1:
					panic(err)
				case 2:
					return []reflect.Value{reflect.Zero(c.pluginType), reflect.ValueOf(&err).Elem()}
				default:
					panic(fmt.Sprintf(" out params num expeced to be 1 or 2, but have: %v", factoryType.NumOut()))
				}
			}
		}
		out := c.newPlugin.Call(maybeConf)
		return convertFactoryOutParam(c.pluginType, factoryType.NumOut(), out)
	}).Interface(), nil
}

// factoryConstructor is abstract constructor of some pluginType interface and it's
// factory functions, using some func newFactory func.
type factoryConstructor struct {
	pluginType reflect.Type
	// newFactory type is func([config <configType>]) (func() (<pluginImpl> [, error])),
	// where configType kind is struct or struct pointer.
	newFactory reflect.Value
}

func (c *factoryConstructor) NewPlugin(maybeConf []reflect.Value) (plugin interface{}, err error) {
	factory, err := c.callNewFactory(maybeConf)
	if err != nil {
		return nil, err
	}
	out := factory.Call(nil)
	plugin = out[0].Interface()
	if len(out) > 1 {
		err, _ = out[1].Interface().(error)
	}
	return
}

func (c *factoryConstructor) NewFactory(factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) (interface{}, error) {
	var maybeConf []reflect.Value
	if getMaybeConf != nil {
		var err error
		maybeConf, err = getMaybeConf()
		if err != nil {
			return nil, err
		}
	}
	factory, err := c.callNewFactory(maybeConf)
	if err != nil {
		return nil, err
	}
	if factory.Type() == factoryType {
		return factory.Interface(), nil
	}
	return reflect.MakeFunc(factoryType, func(in []reflect.Value) []reflect.Value {
		out := factory.Call(nil)
		return convertFactoryOutParam(c.pluginType, factoryType.NumOut(), out)
	}).Interface(), nil
}

func (c *factoryConstructor) callNewFactory(maybeConf []reflect.Value) (factory reflect.Value, err error) {
	factoryAndMaybeErr := c.newFactory.Call(maybeConf)
	if len(factoryAndMaybeErr) > 1 {
		err, _ = factoryAndMaybeErr[1].Interface().(error)
	}
	return factoryAndMaybeErr[0], err
}

// convertFactoryOutParam converts output params of some factory (newFactory) call, to required.
func convertFactoryOutParam(pluginType reflect.Type, numOut int, out []reflect.Value) []reflect.Value {
	switch numOut {
	case 1, 2:
		// OK.
	default:
		panic(fmt.Sprintf("unexpeced out params num: %v; 1 or 2 expected", numOut))
	}
	if out[0].Type() != pluginType {
		// Not plugin, but its implementation.
		impl := out[0]
		out[0] = reflect.New(pluginType).Elem()
		out[0].Set(impl)
	}
	if len(out) < numOut {
		// Registered factory returns no error, but we should.
		out = append(out, reflect.Zero(errorType))
	}
	if numOut < len(out) {
		// Registered factory returns error, but we should not.
		if !out[1].IsNil() {
			panic(out[1].Interface())
		}
		out = out[:1]
	}
	return out
}
