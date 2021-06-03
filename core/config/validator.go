// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"github.com/pkg/errors"
	"gopkg.in/bluesuncorp/validator.v9"
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

func Validate(value interface{}) error {
	return errors.WithStack(defaultValidator.Struct(value))
}

func newValidator() *validator.Validate {
	validate := validator.New()
	validate.SetTagName("validate")
	for _, val := range validations {
		_ = validate.RegisterValidation(val.key, val.val)
	}
	for _, val := range stringValidations {
		_ = validate.RegisterValidation(val.key, StringToAbstractValidation(val.val))
	}
	return validate
}

// RegisterCustom used to set custom validation check hooks on specific types,
// that will be called on such type validation, even if it is nested field.
func RegisterCustom(v CustomValidation, types ...interface{}) (_ struct{}) {
	if len(types) < 1 {
		panic("should be registered for at least one type")
	}
	defaultValidator.RegisterStructValidation(func(sl validator.StructLevel) {
		v(structLevelHandle{sl})
	}, types...)
	return
}

type StringValidation func(value string) bool

// StringToAbstractValidation wraps StringValidation into validator.Func.
func StringToAbstractValidation(sv StringValidation) validator.Func {
	return func(fl validator.FieldLevel) bool {
		if strVal, ok := fl.Field().Interface().(string); ok {
			return sv(strVal)
		}
		return false
	}
}

type ValidateHandle interface {
	Value() interface{}
	ReportError(field, reason string)
}

type CustomValidation func(h ValidateHandle)

type structLevelHandle struct{ validator.StructLevel }

var _ ValidateHandle = structLevelHandle{}

func (sl structLevelHandle) Value() interface{} { return sl.Current().Interface() }
func (sl structLevelHandle) ReportError(field, reason string) {
	sl.StructLevel.ReportError(nil, field, "", reason, "")
}
