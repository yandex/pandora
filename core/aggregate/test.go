// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregate

import (
	"context"
	"sync"

	"github.com/yandex/pandora/core"
)

func NewTest() *Test {
	return &Test{}
}

type Test struct {
	lock    sync.Mutex
	Samples []core.Sample
}

func (t *Test) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}

func (t *Test) Release(s core.Sample) {
	t.lock.Lock()
	t.Samples = append(t.Samples, s)
	t.lock.Unlock()
}
