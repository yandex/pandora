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

func NewConstConf(conf ConstConfig) core.Schedule {
	return NewConst(conf.Ops, conf.Duration)
}

func NewConst(ops float64, duration time.Duration) core.Schedule {
	if ops < 0 {
		ops = 0
	}
	xn := float64(duration) / 1e9 // Seconds.
	n := int64(ops * xn)
	return NewDoAtSchedule(duration, n, constDoAt(ops))
}

func constDoAt(ops float64) func(i int64) time.Duration {
	billionDivOps := 1e9 / ops
	return func(i int64) time.Duration {
		return time.Duration(float64(i) * billionDivOps)
		//return time.Duration(float64(i) * 1e9 / ops)
	}
}
