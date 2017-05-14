// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregate

import (
	"context"

	"github.com/yandex/pandora/core"
)

func NewDiscard() discard {
	return discard{}
}

type discard struct{}

var _ core.Aggregator = discard{}

func (discard) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (discard) Release(core.Sample) {}
