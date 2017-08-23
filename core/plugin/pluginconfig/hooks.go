// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

// Package pluginconfig contains integration plugin with config packages.
// Doing such integration in different package allows to config and plugin packages
// not depend on each other, and set hooks when their are really needed.
package pluginconfig

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/plugin"
	"github.com/yandex/pandora/lib/tag"
)

func AddHooks() {
	config.AddTypeHook(Hook)
	config.AddTypeHook(FactoryHook)
}

const PluginNameKey = "type"

func Hook(f reflect.Type, t reflect.Type, data interface{}) (p interface{}, err error) {
	if !plugin.Lookup(t) {
		return data, nil
	}
	name, fillConf, err := parseConf(t, data)
	if err != nil {
		return
	}
	return plugin.New(t, name, fillConf)
}

func FactoryHook(f reflect.Type, t reflect.Type, data interface{}) (p interface{}, err error) {
	if !plugin.LookupFactory(t) {
		return data, nil
	}
	name, fillConf, err := parseConf(t, data)
	if err != nil {
		return
	}
	return plugin.NewFactory(t, name, fillConf)
}

func parseConf(t reflect.Type, data interface{}) (name string, fillConf func(conf interface{}) error, err error) {
	if tag.Debug {
		zap.L().Debug("Parsing plugin config",
			zap.Stringer("plugin", t),
			zap.Reflect("conf", data),
		)
	}
	confData, err := toStringKeyMap(data)
	if err != nil {
		return
	}
	var names []string
	for key, val := range confData {
		if PluginNameKey == strings.ToLower(key) {
			strVal, ok := val.(string)
			if !ok {
				err = errors.Errorf("%s has non-string value %s", PluginNameKey, val)
				return
			}
			names = append(names, strVal)
			delete(confData, key)
		}
	}
	if len(names) == 0 {
		err = errors.Errorf("plugin %s expected", PluginNameKey)
		return
	}
	if len(names) > 1 {
		err = errors.Errorf("too many %s keys", PluginNameKey)
		return
	}
	name = names[0]
	fillConf = func(conf interface{}) error {
		if tag.Debug {
			zap.L().Debug("Decoding plugin",
				zap.String("name", name),
				zap.Stringer("type", t),
				zap.Stringer("config type", reflect.TypeOf(conf).Elem()),
				zap.String("config data", fmt.Sprint(confData)),
			)
		}
		err := config.DecodeAndValidate(confData, conf)
		if err != nil {
			err = fmt.Errorf("%s %s plugin\n"+
				"%s from %v %s",
				t, name, reflect.TypeOf(conf).Elem(), confData, err)
		}
		return err
	}
	return
}

func toStringKeyMap(data interface{}) (out map[string]interface{}, err error) {
	out, ok := data.(map[string]interface{})
	if ok {
		return
	}
	untypedKeyData, ok := data.(map[interface{}]interface{})
	if !ok {
		err = errors.Errorf("unexpected config type %T: should be map[string or interface{}]interface{}", data)
		return
	}
	out = make(map[string]interface{}, len(untypedKeyData))
	for key, val := range untypedKeyData {
		strKey, ok := key.(string)
		if !ok {
			err = errors.Errorf("unexpected key type %T: %v", key, key)
		}
		out[strKey] = val
	}
	return
}
