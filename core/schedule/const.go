// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"time"

	"github.com/yandex/pandora/core"
)

type ConstConfig struct {
	Ops      float64       `validate:"min=0"`
	Duration time.Duration `validate:"min-time=1ms"`
}

func NewConst(conf ConstConfig) core.Schedule {
	if conf.Ops < 0 {
		conf.Ops = 0
	}
	xn := float64(conf.Duration) / 1e9 // Seconds.
	n := int64(conf.Ops * xn)
	return NewDoAtSchedule(conf.Duration, n, constDoAt(conf.Ops))
}

func constDoAt(ops float64) func(i int64) time.Duration {
	return func(i int64) time.Duration {
		return time.Duration(float64(i) * 1e9 / ops)
	}
}
