// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregator

import (
	"context"
	"sync"

	"a.yandex-team.ru/load/projects/pandora/core"
)

func NewTest() *Test {
	return &Test{}
}

type Test struct {
	lock    sync.Mutex
	samples []core.Sample
}

var _ core.Aggregator = (*Test)(nil)

func (t *Test) Run(ctx context.Context, _ core.AggregatorDeps) error {
	<-ctx.Done()
	return nil
}

func (t *Test) Report(s core.Sample) {
	t.lock.Lock()
	t.samples = append(t.samples, s)
	t.lock.Unlock()
}

func (t *Test) GetSamples() []core.Sample {
	t.lock.Lock()
	s := t.samples
	t.lock.Unlock()
	return s
}
