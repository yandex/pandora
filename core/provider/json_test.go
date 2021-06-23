// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package provider

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/datasource"
)

type testJSONAmmo struct {
	ID   string
	Data string
}

// TODO(skipor): test this in decode provider, not json
func TestDecodeProviderPasses(t *testing.T) {
	input := strings.NewReader(` {"data":"first"} `)
	conf := DefaultJSONProviderConfig()
	conf.Decode.Source = datasource.NewReader(input)
	conf.Decode.Passes = 3
	newAmmo := func() core.Ammo {
		return &testJSONAmmo{}
	}
	provider := NewJSONProvider(newAmmo, conf)
	err := provider.Run(context.Background(), testDeps())
	require.NoError(t, err)

	expected := func(data string) *testJSONAmmo {
		return &testJSONAmmo{Data: data}
	}
	ammo, ok := provider.Acquire()
	require.True(t, ok)
	assert.Equal(t, expected("first"), ammo)

	ammo, ok = provider.Acquire()
	require.True(t, ok)
	assert.Equal(t, expected("first"), ammo)

	ammo, ok = provider.Acquire()
	require.True(t, ok)
	assert.Equal(t, expected("first"), ammo)

	_, ok = provider.Acquire()
	assert.False(t, ok)
}

func TestCustomJSONProvider(t *testing.T) {
	input := strings.NewReader(` {"data":"first"}`)
	conf := DefaultJSONProviderConfig()
	conf.Decode.Source = datasource.NewReader(input)
	conf.Decode.Limit = 1
	newAmmo := func() core.Ammo {
		return &testJSONAmmo{}
	}
	wrapDecoder := func(_ core.ProviderDeps, decoder AmmoDecoder) AmmoDecoder {
		return AmmoDecoderFunc(func(ammo core.Ammo) error {
			err := decoder.Decode(ammo)
			if err != nil {
				return err
			}
			ammo.(*testJSONAmmo).Data += " transformed"
			return nil
		})
	}
	provider := NewCustomJSONProvider(wrapDecoder, newAmmo, conf)
	err := provider.Run(context.Background(), testDeps())
	require.NoError(t, err)
	expected := func(data string) *testJSONAmmo {
		return &testJSONAmmo{Data: data}
	}
	ammo, ok := provider.Acquire()
	require.True(t, ok)
	assert.Equal(t, expected("first transformed"), ammo)

	_, ok = provider.Acquire()
	assert.False(t, ok)
}

func TestJSONProvider(t *testing.T) {
	input := strings.NewReader(` {"data":"first"}
{"data":"second"} `)
	conf := DefaultJSONProviderConfig()
	conf.Decode.Source = datasource.NewReader(input)
	conf.Decode.Limit = 3
	newAmmo := func() core.Ammo {
		return &testJSONAmmo{}
	}
	provider := NewJSONProvider(newAmmo, conf)
	err := provider.Run(context.Background(), testDeps())
	require.NoError(t, err)

	expected := func(data string) *testJSONAmmo {
		return &testJSONAmmo{Data: data}
	}
	ammo, ok := provider.Acquire()
	require.True(t, ok)
	assert.Equal(t, expected("first"), ammo)

	ammo, ok = provider.Acquire()
	require.True(t, ok)
	assert.Equal(t, expected("second"), ammo)

	ammo, ok = provider.Acquire()
	require.True(t, ok)
	assert.Equal(t, expected("first"), ammo)

	_, ok = provider.Acquire()
	assert.False(t, ok)
}

func TestDecoderSimple(t *testing.T) {
	var val struct {
		Data string
	}
	input := strings.NewReader(`{"data":"first"}`)
	decoder := NewJSONAmmoDecoder(input, 512)
	err := decoder.Decode(&val)
	require.NoError(t, err)
	assert.Equal(t, "first", val.Data)

	err = decoder.Decode(&val)
	require.Equal(t, io.EOF, err)
}

func TestDecoderWhitespaces(t *testing.T) {
	var val struct {
		Data string
	}
	input := strings.NewReader(` {"data":"first"}
 {"data":"second"} {"data":"third"} `)
	decoder := NewJSONAmmoDecoder(input, 512)
	err := decoder.Decode(&val)
	require.NoError(t, err)
	assert.Equal(t, "first", val.Data)

	err = decoder.Decode(&val)
	require.NoError(t, err)
	assert.Equal(t, "second", val.Data)

	err = decoder.Decode(&val)
	require.NoError(t, err)
	assert.Equal(t, "third", val.Data)

	err = decoder.Decode(&val)
	require.Equal(t, io.EOF, err)
}

func TestDecoderReset(t *testing.T) {
	val := testJSONAmmo{
		ID: "id",
	}
	input := strings.NewReader(`{"data":"first"}`)
	decoder := NewJSONAmmoDecoder(input, 512)
	err := decoder.Decode(&val)
	require.NoError(t, err)
	assert.Equal(t, "first", val.Data)
	assert.Zero(t, val.ID)
}
