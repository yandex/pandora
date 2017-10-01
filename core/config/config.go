// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"sync"

	"github.com/fatih/structs"
	"github.com/mitchellh/mapstructure"
	"github.com/pkg/errors"
)

const TagName = "config"

// Decodes conf to result. Doesn't zero fields.
func Decode(conf interface{}, result interface{}) error {
	decoder, err := mapstructure.NewDecoder(newDecoderConfig(result))
	if err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(decoder.Decode(conf))
}

func DecodeAndValidate(conf interface{}, result interface{}) error {
	err := Decode(conf, result)
	if err != nil {
		return err
	}
	return Validate(result)
}

type TypeHook mapstructure.DecodeHookFuncType
type KindHook mapstructure.DecodeHookFuncKind

// Returning value allow do `var _ = AddHookType(xxx)`
func AddTypeHook(hook TypeHook) (_ struct{}) {
	addHook(hook)
	return
}

func AddKindHook(hook KindHook) (_ struct{}) {
	addHook(hook)
	return
}

// Map maps with overwrite fields from src to dst.
// if src filed have `map:""` tag, tag value will
// be used as dst field destination.
// src field destinations should be subset of dst fields.
// dst should be struct pointer. src should be struct or struct pointer.
// Example: you need to configure only some subset fields of struct Multi,
// in such case you can from this subset of fields struct Single, decode config
// into it, and map it on Multi.
func Map(dst, src interface{}) {
	conf := &mapstructure.DecoderConfig{
		ErrorUnused: true,
		ZeroFields:  true,
		Result:      dst,
	}
	d, err := mapstructure.NewDecoder(conf)
	if err != nil {
		panic(err)
	}
	s := structs.New(src)
	s.TagName = "map"
	err = d.Decode(s.Map())
	if err != nil {
		panic(err)
	}
}

func newDecoderConfig(result interface{}) *mapstructure.DecoderConfig {
	compileHookOnce.Do(func() {
		compiledHook = mapstructure.ComposeDecodeHookFunc(hooks...)
	})
	return &mapstructure.DecoderConfig{
		DecodeHook:       compiledHook,
		ErrorUnused:      true,
		ZeroFields:       false,
		WeaklyTypedInput: false,
		TagName:          TagName,
		Result:           result,
	}
}

var hooks = []mapstructure.DecodeHookFunc{
	DebugHook,
	mapstructure.StringToTimeDurationHookFunc(),
	StringToURLHook,
	StringToIPHook,
	StringToDataSizeHook,
}

var compiledHook mapstructure.DecodeHookFunc
var compileHookOnce = sync.Once{}

func addHook(hook mapstructure.DecodeHookFunc) {
	if compiledHook != nil {
		panic("all hooks should be added before first decode")
	}
	hooks = append(hooks, hook)
}
