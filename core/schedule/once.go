// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"time"

	"github.com/yandex/pandora/core"
)

type OnceConfig struct {
	N int64 `validate:"min=1"`
}

// NewOnce returns shcedule that emits all passed operation token at start time.
// That is, is scheule for zero duration, unlimited RPS, and conf.N operations.
func NewOnce(conf OnceConfig) core.Schedule {
	return NewDoAtSchedule(0, conf.N, func(i int64) time.Duration {
		return 0
	})
}
