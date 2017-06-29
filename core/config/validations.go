// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"net"
	"reflect"
	"regexp"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/c2h5oh/datasize"
	"gopkg.in/go-playground/validator.v8"
)

func MinTimeValidation(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
	t, min, ok := getTimeForValidation(field.Interface(), param)
	return ok && min <= t
}
func MaxTimeValidation(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
	t, max, ok := getTimeForValidation(field.Interface(), param)
	return ok && t <= max
}

func getTimeForValidation(v interface{}, param string) (actual time.Duration, check time.Duration, ok bool) {
	check, err := time.ParseDuration(param)
	if err != nil {
		return
	}
	actual, ok = v.(time.Duration)
	return
}

func MinSizeValidation(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
	t, min, ok := getSizeForValidation(field.Interface(), param)
	return ok && min <= t
}
func MaxSizeValidation(v *validator.Validate, topStruct reflect.Value, currentStruct reflect.Value, field reflect.Value, fieldtype reflect.Type, fieldKind reflect.Kind, param string) bool {
	t, max, ok := getSizeForValidation(field.Interface(), param)
	return ok && t <= max
}

func getSizeForValidation(v interface{}, param string) (actual, check datasize.ByteSize, ok bool) {
	err := check.UnmarshalText([]byte(param))
	if err != nil {
		return
	}
	actual, ok = v.(datasize.ByteSize)
	return
}

// "host:port" or ":port"
func EndpointStringValidation(value string) bool {
	host, port, err := net.SplitHostPort(value)
	return err == nil &&
		(host == "" || govalidator.IsHost(host)) &&
		govalidator.IsPort(port)
}

// pathRegexp is regexp for at least one path component.
// Valid characters are taken from RFC 3986.
var pathRegexp = regexp.MustCompile(`^(/[a-zA-Z0-9._~!$&'()*+,;=:@%-]+)+$`)

func URLPathStringValidation(value string) bool {
	return pathRegexp.MatchString(value)
}
