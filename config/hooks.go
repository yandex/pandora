// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MLP 2.0
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
	"github.com/facebookgo/stackerr"

	"github.com/yandex/pandora/plugin"
)

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
	if f.Kind() != reflect.Map {
		return nil, stackerr.Newf("%s %v. plugin config should be map", t, data)
	}
	var (
		pluginType string
		confData   map[string]interface{}
	)
	pluginType, confData, err = parsePluginConf(data)
	if err != nil {
		return
	}
	return plugin.New(t, pluginType, func(conf interface{}) error {
		err := DecodeAndValidate(confData, conf)
		if err != nil {
			err = fmt.Errorf("%s %s plugin. %v %s", t, pluginType, confData, err)
		}
		return err
	})
}

func parsePluginConf(data interface{}) (pluginType string, conf map[string]interface{}, err error) {
	conf = toStringKeyMap(data)
	var typeValues []string
	for key, val := range conf {
		if strings.ToLower(key) == "type" {
			strValue, ok := val.(string)
			if !ok {
				err = stackerr.Newf("type has non-string value %s", val)
				return
			}
			typeValues = append(typeValues, strValue)
			delete(conf, key)
		}
	}
	if len(typeValues) == 0 {
		err = stackerr.Newf("plugin type expected")
		return
	}
	if len(typeValues) > 1 {
		err = stackerr.Newf("too many type keys")
		return
	}
	pluginType = typeValues[0]
	return
}

func toStringKeyMap(data interface{}) map[string]interface{} {
	out, ok := data.(map[string]interface{})
	if !ok {
		// map[interface{}]interface{}, where keys really are strings
		// is last valid option. Panic otherwise, like mapstructure do.
		untypedKeyInput := data.(map[interface{}]interface{})
		out = make(map[string]interface{}, len(untypedKeyInput))
		for key, val := range untypedKeyInput {
			strKey := key.(string)
			out[strKey] = val
		}
	}
	return out
}
