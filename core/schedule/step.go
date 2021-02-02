// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"time"

	"github.com/yandex/pandora/core"
)

func NewStep(from, to float64, step int64, duration time.Duration) core.Schedule {
	var nexts []core.Schedule

	if from == to {
		return NewConst(from, duration)
	}

	for i := from; i <= to; i += float64(step) {
		nexts = append(nexts, NewConst(i, duration))
	}

	return NewCompositeConf(CompositeConf{nexts})
}

type StepConfig struct {
	From     float64 `validate:"min=0"`
	To       float64 `validate:"min=0"`
	Step     int64
	Duration time.Duration `validate:"min-time=1ms"`
}

func NewStepConf(conf StepConfig) core.Schedule {
	return NewStep(conf.From, conf.To, conf.Step, conf.Duration)
}
