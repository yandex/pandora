// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/c2h5oh/datasize"
	"github.com/facebookgo/stack"
	"github.com/facebookgo/stackerr"

	"github.com/yandex/pandora/core/plugin"
	"github.com/yandex/pandora/lib/tag"
)

const PluginNameKey = "type"

var InvalidURLError = errors.New("string is not valid URL")

var (
	urlPtrType = reflect.TypeOf(&url.URL{})
	urlType    = reflect.TypeOf(url.URL{})
)

// StringToURLHook converts string to url.URL or *url.URL
func StringToURLHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != urlPtrType && t != urlType {
		return data, nil
	}
	str := data.(string)

	if !govalidator.IsURL(str) { // checks more than url.Parse
		return nil, stackerr.Wrap(InvalidURLError)
	}
	urlPtr, err := url.Parse(str)
	if err != nil {
		return nil, stackerr.Wrap(err)
	}

	if t == urlType {
		return *urlPtr, nil
	}
	return urlPtr, nil
}

var InvalidIPError = errors.New("string is not valid IP")

// StringToIPHook converts string to net.IP
func StringToIPHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != reflect.TypeOf(net.IP{}) {
		return data, nil
	}
	str := data.(string)
	ip := net.ParseIP(str)
	if ip == nil {
		return nil, stackerr.Wrap(InvalidIPError)
	}
	return ip, nil
}

// StringToDataSizeHook converts string to datasize.Single
func StringToDataSizeHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != reflect.TypeOf(datasize.B) {
		return data, nil
	}
	var size datasize.ByteSize
	err := size.UnmarshalText([]byte(data.(string)))
	return size, err
}

func PluginHook(f reflect.Type, t reflect.Type, data interface{}) (p interface{}, err error) {
	if !plugin.Lookup(t) {
		return data, nil
	}
	name, fillConf, err := parseConf(t, data)
	if err != nil {
		return
	}
	return plugin.New(t, name, fillConf)
}

func PluginFactoryHook(f reflect.Type, t reflect.Type, data interface{}) (p interface{}, err error) {
	if !plugin.LookupFactory(t) {
		return data, nil
	}
	name, fillConf, err := parseConf(t, data)
	if err != nil {
		return
	}
	return plugin.NewFactory(t, name, fillConf)
}

// DebugHook used to debug config decode.
func DebugHook(f reflect.Type, t reflect.Type, data interface{}) (p interface{}, err error) {
	p, err = data, nil
	if !tag.Debug {
		return
	}
	callers := stack.Callers(2)
	var decodeCallers int
	for _, caller := range callers {
		if caller.Name == "(*Decoder).decode" {
			decodeCallers++
		}
	}

	offset := strings.Repeat("    ", decodeCallers)
	fmt.Printf("%s %s from %s %v\n", offset, t, f, data)
	return
}

func parseConf(t reflect.Type, data interface{}) (name string, fillConf func(conf interface{}) error, err error) {
	confData, err := toStringKeyMap(data)
	if err != nil {
		return
	}
	var names []string
	for key, val := range confData {
		if PluginNameKey == strings.ToLower(key) {
			strVal, ok := val.(string)
			if !ok {
				err = stackerr.Newf("%s has non-string value %s", PluginNameKey, val)
				return
			}
			names = append(names, strVal)
			delete(confData, key)
		}
	}
	if len(names) == 0 {
		err = stackerr.Newf("plugin %s expected", PluginNameKey)
		return
	}
	if len(names) > 1 {
		err = stackerr.Newf("too many %s keys", PluginNameKey)
		return
	}
	name = names[0]
	fillConf = func(conf interface{}) error {
		if tag.Debug {
			fmt.Printf("Decoding %s %s plugin\n"+"%s from %v",
				t, name, reflect.TypeOf(conf).Elem(), confData)
		}
		err := DecodeAndValidate(confData, conf)
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
		err = stackerr.Newf("unexpected config type %T: should be map[string or interface{}]interface{}", data)
		return
	}
	out = make(map[string]interface{}, len(untypedKeyData))
	for key, val := range untypedKeyData {
		strKey := key.(string)
		out[strKey] = val
	}
	return
}
