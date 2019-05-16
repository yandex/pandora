// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"time"

	"github.com/yandex/pandora/core"
)

// NewOnce returns schedule that emits all passed operation token at start time.
// That is, is schedule for zero duration, unlimited RPS, and n operations.
func NewOnce(n int64) core.Schedule {
	return NewDoAtSchedule(0, n, func(i int64) time.Duration {
		return 0
	})
}

type OnceConfig struct {
	Times int64 `validate:"min=1"` // N is decoded like bool
}

func NewOnceConf(conf OnceConfig) core.Schedule {
	return NewOnce(conf.Times)
}
