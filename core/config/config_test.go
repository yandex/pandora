// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"testing"
	"time"

	"github.com/facebookgo/stack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type M map[string]interface{}

type IPStruct struct {
	Val string `validate:"ip"`
}

func TestDecodeValidate(t *testing.T) {
	var data IPStruct
	err := DecodeAndValidate(M{"val": "192.300.200.100"}, &data)
	assert.Error(t, err)
}

type NoTagStruct struct {
	Val string
}

const testVal = "test"

func TestNilInputIsEmptyInput(t *testing.T) {
	require.NotPanics(t, func() {
		var data NoTagStruct
		err := Decode(nil, &data)
		assert.NoError(t, err)
	})
}

func TestNilResultNotPanic(t *testing.T) {
	require.NotPanics(t, func() {
		err := Decode(M{"val": testVal}, (*NoTagStruct)(nil))
		assert.Error(t, err)
	})
}

func TestFieldNameDecode(t *testing.T) {
	var data NoTagStruct
	err := Decode(M{"val": testVal}, &data)
	require.NoError(t, err)
	assert.Equal(t, testVal, data.Val)
}

type TagStruct struct {
	Val string `config:"valAlias"`
}

func TestTagDecode(t *testing.T) {
	var data TagStruct
	err := Decode(M{"ValAlias": testVal}, &data)
	require.NoError(t, err)
	assert.Equal(t, testVal, data.Val)
}

func TestErrorUnused(t *testing.T) {
	var data NoTagStruct
	err := Decode(M{"val": testVal, "unused": testVal}, &data)
	require.Error(t, err)

	err = Decode(M{"vval": testVal}, &data)
	assert.Error(t, err)
}

func TestNoWeakTypedInput(t *testing.T) {
	var data NoTagStruct
	err := Decode(M{"val": 123}, &data)
	assert.Error(t, err)
}

type TimeoutStruct struct {
	Timeout time.Duration `validate:"min-time=1s,max-time=20m"`
}

func TestValidDurationDecode(t *testing.T) {
	var data TimeoutStruct
	expectedDuration := time.Second * 666

	err := DecodeAndValidate(M{"timeout": "666s"}, &data)
	require.NoError(t, err)
	assert.Equal(t, expectedDuration, data.Timeout)
}

func TestInvalidDurationError(t *testing.T) {
	var data TimeoutStruct

	invalidTimeouts := []string{"ssss", "1ss", "1", "1s1", "1.s", "0x50"}
	for _, invalid := range invalidTimeouts {
		err := DecodeAndValidate(M{"timeout": "ssss"}, &data)
		assert.Error(t, err, "invalid case: ", invalid)
	}
}

type Level1 struct {
	Val1 string
	Val2 string
}

type Level2 struct {
	Val1 Level1
	Val2 Level1
}

func TestNestedStructDecode(t *testing.T) {
	const (
		iniVal1 = "val1"
		iniVal2 = "val2"
		newVal  = "newVal"
	)
	l2 := Level2{
		Level1{
			iniVal1,
			iniVal2,
		},
		Level1{
			iniVal1,
			iniVal2,
		},
	}

	err := DecodeAndValidate(M{
		"val1": M{"val1": newVal},
		"val2": M{"val1": ""},
	}, &l2)
	require.NoError(t, err)
	assert.Equal(t, newVal, l2.Val1.Val1)
	assert.Equal(t, iniVal2, l2.Val1.Val2, "one field in intput, but entire struct rewrited")
	assert.Equal(t, "", l2.Val2.Val1, "zero value not override default")
}

type MultiStrings struct {
	A string
	B string
}

type SingleString struct {
	B string
}

func TestMapFlat(t *testing.T) {
	a := &MultiStrings{}
	Map(a, &SingleString{B: "b"})
	assert.Equal(t, &MultiStrings{B: "b"}, a)

	a = &MultiStrings{A: "a", B: "not b"}
	Map(a, &SingleString{B: "b"})
	assert.Equal(t, &MultiStrings{A: "a", B: "b"}, a)
}

func TestMapRecursive(t *testing.T) {
	type N struct {
		MultiStrings
		A string
	}
	type M struct {
		MultiStrings
	}
	n := &N{MultiStrings: MultiStrings{B: "b"}, A: "a"}
	Map(n, &M{MultiStrings: MultiStrings{A: "a"}})
	assert.Equal(t, &N{A: "a", MultiStrings: MultiStrings{A: "a"}}, n)
}

func TestMapTagged(t *testing.T) {
	type N struct {
		MultiStrings
		A string
	}
	type M struct {
		SomeOtherFieldName MultiStrings `map:"MultiStrings"`
	}
	n := &N{MultiStrings: MultiStrings{B: "b"}, A: "a"}
	Map(n, &M{SomeOtherFieldName: MultiStrings{A: "a"}})
	assert.Equal(t, &N{A: "a", MultiStrings: MultiStrings{A: "a"}}, n)
}

func TestDeltaUpdate(t *testing.T) {
	var l2 Level2
	err := Decode(M{
		"val1": M{"val1": "val1", "val2": "val2"},
		"val2": M{"val1": "val3"},
	}, &l2)
	require.NoError(t, err)
	assert.Equal(t, "val1", l2.Val1.Val1)
	assert.Equal(t, "val2", l2.Val1.Val2)
	assert.Equal(t, "val3", l2.Val2.Val1)
	assert.Equal(t, "", l2.Val2.Val2)

	err = DecodeAndValidate(M{
		"val1": M{"val1": "val4"},
		"val2": M{"val2": "val5"},
	}, &l2)
	require.NoError(t, err)
	assert.Equal(t, "val4", l2.Val1.Val1)
	assert.Equal(t, "val2", l2.Val1.Val2)
	assert.Equal(t, "val3", l2.Val2.Val1)
	assert.Equal(t, "val5", l2.Val2.Val2)
}

func TestNextSquash(t *testing.T) {
	data := &struct {
		Level1 struct {
			Level2 struct {
				Foo string
			} `config:",squash"`
		} `config:",squash"`
	}{}

	defer func() {
		r := recover()
		if r == nil {
			return
		}
		t.Fatalf("panic: %s\n %s", r, stack.Callers(3))
	}()

	err := Decode(M{
		"foo": "baz",
	}, &data)
	require.NoError(t, err)
	assert.Equal(t, "baz", data.Level1.Level2.Foo)
}
