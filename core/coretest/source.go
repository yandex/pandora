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
	"github.com/yandex/pandora/lib/testutil"
)

func AssertSourceEqualStdStream(t *testing.T, expectedPtr **os.File, getSource func() core.DataSource) {
	temp, err := ioutil.TempFile("", "")
	require.NoError(t, err)

	backup := *expectedPtr
	defer func() {
		*expectedPtr = backup
	}()
	*expectedPtr = temp
	const testdata = "abcd"
	_, err = io.WriteString(temp, testdata)
	require.NoError(t, err)

	rc, err := getSource().OpenSource()
	require.NoError(t, err)

	err = rc.Close()
	require.NoError(t, err, "std stream should not be closed")

	_, _ = temp.Seek(0, io.SeekStart)
	data, _ := ioutil.ReadAll(temp)
	assert.Equal(t, testdata, string(data))
}

func AssertSourceEqualFile(t *testing.T, fs afero.Fs, filename string, source core.DataSource) {
	const testdata = "abcd"
	_ = afero.WriteFile(fs, filename, []byte(testdata), 0644)

	rc, err := source.OpenSource()
	require.NoError(t, err)

	data := testutil.ReadString(t, rc)
	err = rc.Close()
	require.NoError(t, err)

	assert.Equal(t, testdata, data)
}
