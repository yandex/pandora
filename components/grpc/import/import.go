package example

import (
	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/guns/grpc"
	"github.com/yandex/pandora/components/guns/grpc/scenario"
	"github.com/yandex/pandora/components/providers/grpc/grpcjson"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
)

func Import(fs afero.Fs) {

	register.Provider("grpc/json", func(conf grpcjson.Config) core.Provider {
		return grpcjson.NewProvider(fs, conf)
	})

	register.Gun("grpc", grpc.NewGun, grpc.DefaultGunConfig)
	register.Gun("grpc/scenario", scenario.NewGun, scenario.DefaultGunConfig)
}
