// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"context"
	"net/http"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregate/netsample"
)

//go:generate mockery -name=Ammo -case=underscore -outpkg=ammomock

// Ammo ammo interface for http based guns.
// http ammo providers should produce ammo that implements Ammo.
// http guns should use convert ammo to Ammo, not to specific implementation.
// Returned request have
type Ammo interface {
	// TODO(skipor): instead of sample use it wrapper with httptrace and more usable interface.
	Request() (*http.Request, *netsample.Sample)
	// Id unique ammo id. Usually equals to ammo num got from provider.
	Id() int
}

type Gun interface {
	Shoot(context.Context, Ammo)
	Bind(netsample.Aggregator)
}

func WrapGun(g Gun) core.Gun {
	if g == nil {
		return nil
	}
	return &gunWrapper{g}
}

type gunWrapper struct{ Gun }

func (g *gunWrapper) Shoot(ctx context.Context, ammo core.Ammo) {
	g.Gun.Shoot(ctx, ammo.(Ammo))
}

func (g *gunWrapper) Bind(a core.Aggregator) {
	g.Gun.Bind(netsample.UnwrapAggregator(a))
}
