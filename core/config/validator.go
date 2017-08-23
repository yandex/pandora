// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"reflect"

	"github.com/pkg/errors"

	"gopkg.in/go-playground/validator.v8"
)

var validations = []struct {
	key string
	val validator.Func
}{
	{"min-time", MinTimeValidation},
	{"max-time", MaxTimeValidation},
	{"min-size", MinSizeValidation},
	{"max-size", MaxSizeValidation},
}

var stringValidations = []struct {
	key string
	val StringValidation
}{
	{"endpoint", EndpointStringValidation},
	{"url-path", URLPathStringValidation},
}

var defaultValidator = newValidator()

type validate struct {
	V validator.Validate
}

func (v *validate) Validate(value interface{}) error {
	return errors.WithStack(v.V.Struct(value))
}

func newValidator() *validate {
	config := &validator.Config{TagName: "validate"}
	v := *validator.New(config)
	for _, val := range validations {
		v.RegisterValidation(val.key, val.val)
	}
	for _, val := range stringValidations {
		v.RegisterValidation(val.key, StringToAbstractValidation(val.val))
	}
	return &validate{v}
}

func Validate(value interface{}) error {
	return defaultValidator.Validate(value)
}

type StringValidation func(value string) bool

// StringToAbstractValidation wraps StringValidation into validator.Func.
func StringToAbstractValidation(sv StringValidation) validator.Func {
	return func(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
		if strVal, ok := field.Interface().(string); ok {
			return sv(strVal)
		}
		return false
	}
}
