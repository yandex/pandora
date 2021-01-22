// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type WithURLString struct {
	URL string `validate:"required,url"`
}

func TestValidateURL(t *testing.T) {
	require.NoError(t, Validate(&WithURLString{"http://yandex.ru/"}))

	err := Validate(&WithURLString{"http://yandex.ru/%zz"})
	require.Error(t, err)

	err = Validate(&WithURLString{})
	assert.Error(t, err)
}

type Multi struct {
	A int `validate:"min=1"`
	B int `validate:"min=2"`
}

type Single struct {
	X int `validate:"max=0,min=10"`
}

type Nested struct {
	A Multi
}

func TestValidateOK(t *testing.T) {
	assert.NoError(t, Validate(&Multi{1, 2}))
}

func TestValidateError(t *testing.T) {
	err := Validate(&Multi{0, 2})
	require.Error(t, err)

	err = Validate(&Multi{0, 0})
	require.Error(t, err)

	err = Validate(&Single{5})
	assert.Error(t, err)
}

func TestNestedError(t *testing.T) {
	c := &Nested{
		Multi{0, 0},
	}
	require.Error(t, Validate(c.A))
	err := Validate(c)
	assert.Error(t, err)
}

func TestValidateUnsupported(t *testing.T) {
	err := Validate(1)
	assert.Error(t, err)
}

type D struct {
	Val string `validate:"invalidNameXXXXXXX=1"`
}

func TestValidateInvalidValidatorName(t *testing.T) {
	require.Panics(t, func() {
		_ = Validate(&D{"test"})
	})
}

func TestCustom(t *testing.T) {
	defer func() {
		defaultValidator = newValidator()
	}()
	type custom struct{ fail bool }
	RegisterCustom(func(h ValidateHandle) {
		if h.Value().(custom).fail {
			h.ReportError("fail", "should be false")
		}
	}, custom{})
	assert.NoError(t, Validate(&custom{fail: false}))
	assert.Error(t, Validate(&custom{fail: true}))
}
