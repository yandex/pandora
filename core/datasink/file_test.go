// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package datasink

import (
	"os"
	"testing"

	"github.com/spf13/afero"

	"a.yandex-team.ru/load/projects/pandora/core/coretest"
)

func TestFileSink(t *testing.T) {
	const filename = "/xxx/yyy"
	fs := afero.NewMemMapFs()
	sink := NewFile(fs, FileConfig{Path: filename})
	coretest.AssertSinkEqualFile(t, fs, filename, sink)
}

func TestStdout(t *testing.T) {
	coretest.AssertSinkEqualStdStream(t, &os.Stdout, NewStdout)
}

func TestStderr(t *testing.T) {
	coretest.AssertSinkEqualStdStream(t, &os.Stderr, NewStderr)
}
