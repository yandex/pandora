// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package example

import (
	"github.com/yandex/pandora/components/example"
	"github.com/yandex/pandora/core/register"
)

func Import() {
	register.Provider("example", example.NewProvider, example.DefaultProviderConfig)
	register.Gun("example", example.NewGun)
}
