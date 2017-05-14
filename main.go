// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package main

import (
	"github.com/spf13/afero"

	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/components/example/import"
	"github.com/yandex/pandora/components/phttp/import"
	"github.com/yandex/pandora/core/import"
)

func main() {
	// CLI don't know anything about components initially.
	// All extpoints constructors and default configurations should be registered, before CLI run.
	fs := afero.NewOsFs()
	core.Import(fs)

	// Components should not write anything to files.
	readOnlyFs := afero.NewReadOnlyFs(fs)
	example.Import()
	phttp.Import(readOnlyFs)

	cli.Run()
}
