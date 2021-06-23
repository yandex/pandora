// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package example

import (
	"github.com/yandex/pandora/components/grpc"
	"github.com/yandex/pandora/core"
	coreimport "github.com/yandex/pandora/core/import"
	"github.com/yandex/pandora/core/register"
)

func Import() {
	coreimport.RegisterCustomJSONProvider("grpc/json", func() core.Ammo { return &grpc.Ammo{} })

	register.Gun("grpc", grpc.NewGun, func() grpc.GunConfig {
		return grpc.GunConfig{
			Target: "default target",
		}
	})
}
