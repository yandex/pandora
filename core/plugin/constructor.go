// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import "reflect"

type constructor interface {
	NewPlugin(maybeConf []reflect.Value) (plugin interface{}, err error)
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
	// FIXME: if no config and same type return newPlugin interface
	// FIXME: support no error returning factories
	return reflect.MakeFunc(factoryType, func(in []reflect.Value) (out []reflect.Value) {
		var conf []reflect.Value
		if getMaybeConf != nil {
			var err error
			conf, err = getMaybeConf()
			if err != nil {
				return []reflect.Value{reflect.Zero(c.pluginType), reflect.ValueOf(&err).Elem()}
			}
		}
		out = c.newPlugin.Call(conf)
		if out[0].Type() != c.pluginType {
			// Not plugin, but its implementation.
			impl := out[0]
			out[0] = reflect.New(c.pluginType).Elem()
			out[0].Set(impl)
		}

		if len(out) < 2 {
			// Registered newPlugin can return no error, but we should.
			out = append(out, reflect.Zero(errorType))
		}
		return
	}).Interface(), nil
}
