package main

import (
	"github.com/spf13/afero"
	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/components/answ"
	grpc "github.com/yandex/pandora/components/grpc/import"
	"github.com/yandex/pandora/components/guns"
	phttp "github.com/yandex/pandora/components/phttp/import"
	coreimport "github.com/yandex/pandora/core/import"
)

func main() {
	// CLI don't know anything about components initially.
	// All extpoints constructors and default configurations should be registered, before CLI run.
	fs := afero.NewOsFs()
	coreimport.Import(fs)
	phttp.Import(fs)
	grpc.Import(fs)
	guns.Import(fs)
	answ.Import()

	cli.Run()
}
