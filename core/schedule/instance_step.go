// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"time"

	"github.com/yandex/pandora/core"
)

func NewInstanceStep(from, to int64, step int64, stepDuration time.Duration) core.Schedule {
	var nexts []core.Schedule
	nexts = append(nexts, NewOnce(from))

	for i := from + step; i <= to; i += step {
		nexts = append(nexts, NewConst(0, stepDuration))
		nexts = append(nexts, NewOnce(step))
	}

	return NewCompositeConf(CompositeConf{nexts})
}

type InstanceStepConfig struct {
	From         int64         `validate:"min=0"`
	To           int64         `validate:"min=0"`
	Step         int64         `validate:"min=1"`
	StepDuration time.Duration `validate:"min-time=1ms"`
}

func NewInstanceStepConf(conf InstanceStepConfig) core.Schedule {
	return NewInstanceStep(conf.From, conf.To, conf.Step, conf.StepDuration)
}
