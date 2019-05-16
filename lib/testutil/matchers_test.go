// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlakyPassed(t *testing.T) {
	var run int
	RunFlaky(t, func(t TestingT) {
		run++
		assert.True(t, run >= 3)
	})
}

func TestFlakyPanic(t *testing.T) {
	var run int
	RunFlaky(t, func(t TestingT) {
		run++
		require.True(t, run >= 3)
	})
}
