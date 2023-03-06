// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package example

import (
	"github.com/spf13/afero"
	"github.com/yandex/pandora/components/guns/grpc"
	"github.com/yandex/pandora/components/providers/grpc/grpcjson"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
)

func Import(fs afero.Fs) {

	register.Provider("grpc/json", func(conf grpcjson.Config) core.Provider {
		return grpcjson.NewProvider(fs, conf)
	})

	register.Gun("grpc", grpc.NewGun, func() grpc.GunConfig {
		return grpc.GunConfig{
			Target: "default target",
		}
	})
}
