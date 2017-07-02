// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"github.com/spf13/afero"

	. "github.com/yandex/pandora/components/phttp"
	"github.com/yandex/pandora/components/phttp/ammo/simple/jsonline"
	"github.com/yandex/pandora/components/phttp/ammo/simple/raw"
	"github.com/yandex/pandora/components/phttp/ammo/simple/uri"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/register"
)

func Import(fs afero.Fs) {
	register.Provider("jsonline", func(conf jsonline.Config) core.Provider {
		return jsonline.NewProvider(fs, conf)
	})

	register.Provider("uri", func(conf uri.Config) core.Provider {
		return uri.NewProvider(fs, conf)
	})

	register.Provider("raw", func(conf raw.Config) core.Provider {
		return raw.NewProvider(fs, conf)
	})

	register.Gun("http", func(conf HTTPGunConfig) core.Gun {
		return WrapGun(NewHTTPGun(conf))
	}, NewDefaultHTTPGunConfig)

	register.Gun("connect", func(conf ConnectGunConfig) core.Gun {
		return WrapGun(NewConnectGun(conf))
	}, NewDefaultConnectGunConfig)
}
