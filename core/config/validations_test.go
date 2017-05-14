// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package config

import (
	"testing"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/stretchr/testify/assert"
)

type WithDuration struct {
	T time.Duration `validate:"min-time=10ms,max-time=100ms"`
}

func TestValidateTimeDuration(t *testing.T) {
	assert.NoError(t, Validate(&WithDuration{50 * time.Millisecond}))

	assert.Error(t, Validate(&WithDuration{5 * time.Millisecond}))
	assert.Error(t, Validate(&WithDuration{500 * time.Millisecond}))
}

type WithSize struct {
	T datasize.ByteSize `validate:"min-size=10kb,max-size=10M"`
}

func TestValidateByteSize(t *testing.T) {
	assert.NoError(t, Validate(&WithSize{50 * datasize.KB}))

	assert.Error(t, Validate(&WithSize{5 * datasize.KB}))
	assert.Error(t, Validate(&WithSize{500 * datasize.MB}))
}

type WithEndpoint struct {
	Endpoint string `validate:"required,endpoint"`
}

func TestValidateEndpoint(t *testing.T) {
	assert.NoError(t, Validate(&WithEndpoint{"192.168.0.1:9999"}))
	assert.NoError(t, Validate(&WithEndpoint{"localhost:9999"}))
	assert.NoError(t, Validate(&WithEndpoint{":9999"}))

	assert.Error(t, Validate(&WithEndpoint{"aaaaa"}))
}

type WithURLPath struct {
	Path string `validate:"url-path"`
}

func TestValidatePath(t *testing.T) {
	assert.NoError(t, Validate(&WithURLPath{"/some"}))
	assert.NoError(t, Validate(&WithURLPath{"/just/some/path"}))
	assert.NoError(t, Validate(&WithURLPath{"/just@/some()/+@!="}))
	assert.Error(t, Validate(&WithURLPath{""}))
	assert.Error(t, Validate(&WithURLPath{"/"}))
	assert.Error(t, Validate(&WithURLPath{"/just/"}))
	assert.Error(t, Validate(&WithURLPath{"/just//some"}))
	assert.Error(t, Validate(&WithURLPath{"/just/some?"}))
}
