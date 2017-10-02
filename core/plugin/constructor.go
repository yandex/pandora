// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"fmt"
	"reflect"
)

// implConstructor interface representing ability to create some plugin interface
// implementations and is't factory functions. Usually it wraps some newPlugin or newFactory function, and
// use is as implementation creator.
// implConstructor expects, that caller pass correct maybeConf value, that can be
// passed to underlying implementation creator.
type implConstructor interface {
	// NewPlugin constructs plugin implementation.
	NewPlugin(maybeConf []reflect.Value) (plugin interface{}, err error)
	// getMaybeConf may be nil, if no config required.
	// Underlying implementation creator may require new config for every instance create.
	// If so, then getMaybeConf will be called on every instance create. Otherwise, only once.
	NewFactory(factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) (pluginFactory interface{}, err error)
}

func newImplConstructor(pluginType reflect.Type, constructor interface{}) implConstructor {
	constructorType := reflect.TypeOf(constructor)
	expect(constructorType.Kind() == reflect.Func, "plugin constructor should be func")
	expect(constructorType.NumOut() >= 1,
		"plugin constructor should return plugin implementation as first output parameter")
	if constructorType.Out(0).Kind() == reflect.Func {
		return newFactoryConstructor(pluginType, constructor)
	}
	return newPluginConstructor(pluginType, constructor)
}

func newPluginConstructor(pluginType reflect.Type, newPlugin interface{}) *pluginConstructor {
	expectPluginConstructor(pluginType, reflect.TypeOf(newPlugin), true)
	return &pluginConstructor{pluginType, reflect.ValueOf(newPlugin)}
}

// pluginConstructor use newPlugin func([config <configType>]) (<pluginImpl> [, error])
// to construct plugin implementations
type pluginConstructor struct {
	pluginType reflect.Type
	newPlugin  reflect.Value
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
		return convertFactoryOutParams(c.pluginType, factoryType.NumOut(), out)
	}).Interface(), nil
}

// factoryConstructor use newFactory func([config <configType>]) (func() (<pluginImpl>[, error])[, error)
// to construct plugin implementations.
type factoryConstructor struct {
	pluginType reflect.Type
	newFactory reflect.Value
}

func newFactoryConstructor(pluginType reflect.Type, newFactory interface{}) *factoryConstructor {
	newFactoryType := reflect.TypeOf(newFactory)
	expect(newFactoryType.Kind() == reflect.Func, "factory constructor should be func")
	expect(newFactoryType.NumIn() <= 1, "factory constructor should accept config or nothing")

	expect(1 <= newFactoryType.NumOut() && newFactoryType.NumOut() <= 2,
		"factory constructor should return factory, and optionally error")
	if newFactoryType.NumOut() == 2 {
		expect(newFactoryType.Out(1) == errorType, "factory constructor should have no second return value, or it should be error")
	}
	factoryType := newFactoryType.Out(0)
	expectPluginConstructor(pluginType, factoryType, false)
	return &factoryConstructor{pluginType, reflect.ValueOf(newFactory)}
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
		return convertFactoryOutParams(c.pluginType, factoryType.NumOut(), out)
	}).Interface(), nil
}

func (c *factoryConstructor) callNewFactory(maybeConf []reflect.Value) (factory reflect.Value, err error) {
	factoryAndMaybeErr := c.newFactory.Call(maybeConf)
	if len(factoryAndMaybeErr) > 1 {
		err, _ = factoryAndMaybeErr[1].Interface().(error)
	}
	return factoryAndMaybeErr[0], err
}

// expectPluginConstructor checks type expectations common for newPlugin, and factory, returned from newFactory.
func expectPluginConstructor(pluginType, factoryType reflect.Type, configAllowed bool) {
	expect(factoryType.Kind() == reflect.Func, "plugin constructor should be func")
	if configAllowed {
		expect(factoryType.NumIn() <= 1, "plugin constructor should accept config or nothing")
	} else {
		expect(factoryType.NumIn() == 0, "plugin constructor returned from newFactory, shouldn't accept any arguments")
	}
	expect(1 <= factoryType.NumOut() && factoryType.NumOut() <= 2,
		"plugin constructor should return plugin implementation, and optionally error")
	pluginImplType := factoryType.Out(0)
	expect(pluginImplType.Implements(pluginType), "plugin constructor should implement plugin interface")
	if factoryType.NumOut() == 2 {
		expect(factoryType.Out(1) == errorType, "plugin constructor should have no second return value, or it should be error")
	}
}

// convertFactoryOutParams converts output params of some factory (newPlugin) call to required.
func convertFactoryOutParams(pluginType reflect.Type, numOut int, out []reflect.Value) []reflect.Value {
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
