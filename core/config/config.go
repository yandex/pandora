package config

import (
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

// Map maps with overwrite fields from src to dst.
// if src filed have `map:""` tag, tag value will
// be used as dst field destination.
// src field destinations should be subset of dst fields.
// dst should be struct pointer. src should be struct or struct pointer.
// Example: you need to configure only some subset fields of struct Multi,
// in such case you can from this subset of fields struct Single, decode config
// into it, and map it on Multi.
func Map(dst, src interface{}) {
	// dst and src conf for compatibility with old fatih/structs.
	// src map from "map:" tags -> tmp -> map to "mapstructure:" tags in dst
	dstConf := &mapstructure.DecoderConfig{
		ErrorUnused: true,
		ZeroFields:  true,
		Result:      dst,
	}
	d, err := mapstructure.NewDecoder(dstConf)
	if err != nil {
		panic(err)
	}

	tmp := make(map[string]interface{})
	srcConf := &mapstructure.DecoderConfig{
		ErrorUnused: true,
		ZeroFields:  true,
		Result:      &tmp,
		TagName:     "map",
	}
	s, err := mapstructure.NewDecoder(srcConf)
	if err != nil {
		panic(err)
	}

	err = s.Decode(src)
	if err != nil {
		panic(err)
	}

	err = d.Decode(tmp)
	if err != nil {
		panic(err)
	}
}

func newDecoderConfig(result interface{}) *mapstructure.DecoderConfig {
	compileHooks()
	return &mapstructure.DecoderConfig{
		DecodeHook:       compiledHook,
		ErrorUnused:      true,
		ZeroFields:       false,
		WeaklyTypedInput: false,
		TagName:          TagName,
		Result:           result,
	}
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

func DefaultHooks() []mapstructure.DecodeHookFunc {
	return []mapstructure.DecodeHookFunc{
		VariableInjectHook,
		DebugHook,
		TextUnmarshallerHook,
		mapstructure.StringToTimeDurationHookFunc(),
		StringToURLHook,
		StringToIPHook,
		StringToDataSizeHook,
	}
}

func GetHooks() []mapstructure.DecodeHookFunc {
	return hooks
}
func SetHooks(h []mapstructure.DecodeHookFunc) {
	hooks = h
	onHooksModify()
}

var (
	hooks            = DefaultHooks()
	hooksNeedCompile = true
	compiledHook     mapstructure.DecodeHookFunc
)

func addHook(hook mapstructure.DecodeHookFunc) {
	hooks = append(hooks, hook)
	onHooksModify()
}

func onHooksModify() {
	hooksNeedCompile = true
}

func compileHooks() {
	if hooksNeedCompile {
		compiledHook = mapstructure.ComposeDecodeHookFunc(hooks...)
		hooksNeedCompile = false
	}
}
