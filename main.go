// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package main

import (
	"github.com/spf13/afero"

	"a.yandex-team.ru/load/projects/pandora/cli"
	example "a.yandex-team.ru/load/projects/pandora/components/example/import"
	grpc "a.yandex-team.ru/load/projects/pandora/components/grpc/import"
	phttp "a.yandex-team.ru/load/projects/pandora/components/phttp/import"
	coreimport "a.yandex-team.ru/load/projects/pandora/core/import"
)

func main() {
	// CLI don't know anything about components initially.
	// All extpoints constructors and default configurations should be registered, before CLI run.
	fs := afero.NewOsFs()
	coreimport.Import(fs)
	phttp.Import(fs)
	example.Import()
	grpc.Import()

	cli.Run()
}
