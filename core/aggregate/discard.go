// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregate

import (
	"context"

	"github.com/yandex/pandora/core"
)

// NewDiscard returns Aggregator that just throws reported ammo away.
func NewDiscard() core.Aggregator {
	return discard{}
}

type discard struct{}

func (discard) Run(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (discard) Report(core.Sample) {}
