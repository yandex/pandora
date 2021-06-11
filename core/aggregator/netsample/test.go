// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package netsample

import (
	"context"

	"a.yandex-team.ru/load/projects/pandora/core"
)

type TestAggregator struct {
	Samples []*Sample
}

var _ Aggregator = &TestAggregator{}

func (t *TestAggregator) Run(ctx context.Context, _ core.AggregatorDeps) error {
	<-ctx.Done()
	return nil
}

func (t *TestAggregator) Report(s *Sample) {
	t.Samples = append(t.Samples, s)
}
