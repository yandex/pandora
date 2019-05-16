// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"math"
	"time"

	"github.com/yandex/pandora/core"
)

func NewLine(from, to float64, duration time.Duration) core.Schedule {
	if from == to {
		return NewConst(from, duration)
	}
	a := (to - from) / float64(duration/1e9)
	b := from
	xn := float64(duration) / 1e9
	n := int64(a*xn*xn/2 + b*xn)
	return NewDoAtSchedule(duration, n, lineDoAt(a, b))
}

type LineConfig struct {
	From     float64       `validate:"min=0"`
	To       float64       `validate:"min=0"`
	Duration time.Duration `validate:"min-time=1ms"`
}

func NewLineConf(conf LineConfig) core.Schedule {
	return NewLine(conf.From, conf.To, conf.Duration)
}

// x - duration from 0 to max.
// RPS(x) = a * x + b // Line RPS schedule.
// Number of shots from 0 to x = integral(RPS) from 0 to x = (a*x^2)/2 + b*x
// Has shoot i. When it should be? i = (a*x^2)/2 + b*x => x = (sqrt(2*a*i + b^2) - b) / a
func lineDoAt(a, b float64) func(i int64) time.Duration {
	// Some common calculations.
	twoA := 2 * a
	bSquare := b * b
	bilionDivA := 1e9 / a
	return func(i int64) time.Duration {
		//return time.Duration((math.Sqrt(2*a*float64(i)+b*b) - b) * 1e9 / a)
		return time.Duration((math.Sqrt(twoA*float64(i)+bSquare) - b) * bilionDivA)
	}
}
