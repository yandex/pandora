// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package example

import (
	"a.yandex-team.ru/load/projects/pandora/components/example"
	"a.yandex-team.ru/load/projects/pandora/core/register"
)

func Import() {
	register.Provider("example", example.NewProvider, example.DefaultProviderConfig)
	register.Gun("example", example.NewGun)
}
