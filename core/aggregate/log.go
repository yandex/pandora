// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregate

import (
	"context"
	"log"

	"github.com/yandex/pandora/core"
)

func NewLog() core.Aggregator {
	return &logging{make(chan core.Sample, 128)}
}

type logging struct {
	sink chan core.Sample
}

func (l *logging) Report(sample core.Sample) {
	l.sink <- sample
}

func (l *logging) Run(ctx context.Context) error {
loop:
	for {
		select {
		case sample := <-l.sink:
			l.handle(sample)
		case <-ctx.Done():
			break loop
		}
	}
	for {
		// Context is done, but we should read all data from sink.
		select {
		case r := <-l.sink:
			l.handle(r)
		default:
			return nil
		}
	}
}

func (l *logging) handle(sample core.Sample) {
	log.Println(sample)
}
