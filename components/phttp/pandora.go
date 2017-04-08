// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"net/http"

	"github.com/yandex/pandora/core/aggregate"
)

//go:generate mockery -name=Ammo -case=underscore -outpkg=ammomocks

// Ammo ammo interface for http based guns.
// http ammo providers should produce ammo that implements Ammo.
// http guns should use convert ammo to Ammo, not to specific implementation.
// Returned request have
type Ammo interface {
	// TODO(skipor): instead of sample use some more usable interface.
	Request() (*http.Request, *aggregate.Sample)
}
