// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"encoding"
	stderrors "errors"
	"fmt"
	"net"
	"net/url"
	"reflect"

	"github.com/asaskevich/govalidator"
	"github.com/c2h5oh/datasize"
	"github.com/facebookgo/stack"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/yandex/pandora/lib/tag"
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
		return nil, errors.WithStack(InvalidURLError)
	}
	urlPtr, err := url.Parse(str)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if t == urlType {
		return *urlPtr, nil
	}
	return urlPtr, nil
}

var ErrInvalidIP = stderrors.New("string is not valid IP")

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
		return nil, errors.WithStack(ErrInvalidIP)
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

var textUnmarshallerType = func() reflect.Type {
	var ptr *encoding.TextUnmarshaler
	return reflect.TypeOf(ptr).Elem()
}

func TextUnmarshallerHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t.Implements(textUnmarshallerType()) {
		val := reflect.Zero(t)
		if t.Kind() == reflect.Ptr {
			val = reflect.New(t.Elem())
		}
		err := unmarhsallText(val, data)
		return val.Interface(), err
	}
	if reflect.PtrTo(t).Implements(textUnmarshallerType()) {
		val := reflect.New(t)
		err := unmarhsallText(val, data)
		return val.Elem().Interface(), err
	}
	return data, nil
}

func unmarhsallText(v reflect.Value, data interface{}) error {
	unmarshaller := v.Interface().(encoding.TextUnmarshaler)
	// unmarshaller.UnmarshalText([]byte(data.(string)))
	err := unmarshaller.UnmarshalText([]byte(data.(string)))
	return err
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
	zap.L().Debug("Config decode",
		zap.Int("depth", decodeCallers),
		zap.Stringer("type", t),
		zap.Stringer("from", f),
		zap.String("data", fmt.Sprint(data)),
	)
	return
}
