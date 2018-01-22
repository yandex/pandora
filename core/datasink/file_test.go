// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package datasink

import (
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileSink(t *testing.T) {
	const filename = "/xxx/yyy"
	fs := afero.NewMemMapFs()
	afero.WriteFile(fs, filename, []byte("should be truncated"), 644)

	wc, err := NewFileSink(fs, FileSinkConfig{Path: filename}).OpenSink()
	require.NoError(t, err)

	const testdata = "abcd"

	_, err = io.WriteString(wc, testdata)
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	data, err := afero.ReadFile(fs, filename)
	require.NoError(t, err)

	assert.Equal(t, testdata, string(data))

	NewFileSink(fs, FileSinkConfig{
		Path: filename,
	})

}

func TestStdout(t *testing.T) {
	temp, err := ioutil.TempFile("", "")
	require.NoError(t, err)

	stdout := os.Stdout
	defer func() {
		os.Stdout = stdout
	}()
	os.Stdout = temp
	const testdata = "abcd"

	wc, err := NewStdoutSink().OpenSink()
	require.NoError(t, err)

	_, err = io.WriteString(wc, testdata)
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	temp.Seek(0, io.SeekStart)
	data, err := ioutil.ReadAll(temp)
	assert.Equal(t, testdata, string(data))
}

func TestStderr(t *testing.T) {
	temp, err := ioutil.TempFile("", "")
	require.NoError(t, err)

	stderr := os.Stderr
	defer func() {
		os.Stderr = stderr
	}()
	os.Stderr = temp
	const testdata = "abcd"

	wc, err := NewStderrSink().OpenSink()
	require.NoError(t, err)

	_, err = io.WriteString(wc, testdata)
	require.NoError(t, err)

	err = wc.Close()
	require.NoError(t, err)

	temp.Seek(0, io.SeekStart)
	data, err := ioutil.ReadAll(temp)
	assert.Equal(t, testdata, string(data))
}
