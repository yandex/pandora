// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coretest

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/pandora/core"
)

func AssertSinkEqualStdStream(t *testing.T, expectedPtr **os.File, getSink func() core.DataSink) {
	temp, err := ioutil.TempFile("", "")
	require.NoError(t, err)

	backup := *expectedPtr
	defer func() {
		*expectedPtr = backup
	}()
	*expectedPtr = temp
	const testdata = "abcd"

	wc, err := getSink().OpenSink()
	require.NoError(t, err)

	_, err = io.WriteString(wc, testdata)
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	_, _ = temp.Seek(0, io.SeekStart)
	data, _ := ioutil.ReadAll(temp)
	assert.Equal(t, testdata, string(data))
}

func AssertSinkEqualFile(t *testing.T, fs afero.Fs, filename string, sink core.DataSink) {
	_ = afero.WriteFile(fs, filename, []byte("should be truncated"), 0644)

	wc, err := sink.OpenSink()
	require.NoError(t, err)

	const testdata = "abcd"

	_, err = io.WriteString(wc, testdata)
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	data, err := afero.ReadFile(fs, filename)
	require.NoError(t, err)

	assert.Equal(t, testdata, string(data))
}
