// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package testutil

import (
	"io"
	"io/ioutil"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ReadString(t TestingT, r io.Reader) string {
	data, err := ioutil.ReadAll(r)
	require.NoError(t, err)
	return string(data)
}

func ReadFileString(t TestingT, fs afero.Fs, name string) string {
	getHelper(t).Helper()
	data, err := afero.ReadFile(fs, name)
	require.NoError(t, err)
	return string(data)

}

func AssertFileEqual(t TestingT, fs afero.Fs, name string, expected string) {
	getHelper(t).Helper()
	actual := ReadFileString(t, fs, name)
	assert.Equal(t, expected, actual)
}
