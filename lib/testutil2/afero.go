// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package testutil2

import (
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ReadFileString(t TestingT, fs afero.Fs, name string) string {
	data, err := afero.ReadFile(fs, name)
	require.NoError(t, err)
	return string(data)

}

func AssertFileEqual(t TestingT, fs afero.Fs, name string, expected string) {
	actual := ReadFileString(t, fs, name)
	assert.Equal(t, expected, actual)
}
