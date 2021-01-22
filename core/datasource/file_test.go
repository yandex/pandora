// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package datasource

import (
	"os"
	"testing"

	"github.com/yandex/pandora/core/coretest"
	"github.com/spf13/afero"
)

func TestFileSource(t *testing.T) {
	const filename = "/xxx/yyy"
	fs := afero.NewMemMapFs()
	source := NewFile(fs, FileConfig{Path: filename})
	coretest.AssertSourceEqualFile(t, fs, filename, source)
}

func TestStdin(t *testing.T) {
	coretest.AssertSourceEqualStdStream(t, &os.Stdout, NewStdin)
}
