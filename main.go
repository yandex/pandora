// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package main

import (
	"github.com/spf13/afero"

	"github.com/yandex/pandora/cli"
	example "github.com/yandex/pandora/components/example/import"
	phttp "github.com/yandex/pandora/components/phttp/import"
	coreimport "github.com/yandex/pandora/core/import"
)

func main() {
	// CLI don't know anything about components initially.
	// All extpoints constructors and default configurations should be registered, before CLI run.
	fs := afero.NewOsFs()
	coreimport.Import(fs)
	phttp.Import(fs)
	example.Import()

	cli.Run()
}
