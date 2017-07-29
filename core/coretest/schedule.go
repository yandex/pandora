// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coretest

import (
	"time"

	"github.com/onsi/gomega"

	"github.com/yandex/pandora/core"
)

func ExpectScheduleNexts(sched core.Schedule, nexts ...time.Duration) {
	actualNexts := StartAndDrainSchedule(sched)
	gomega.ExpectWithOffset(1, actualNexts).To(gomega.Equal(nexts))
}

// StartAndDrainSchedule starts shcedule and takes all tokens from it.
// Returns all tokens and finish time relative to start
func StartAndDrainSchedule(sched core.Schedule) []time.Duration {
	start := time.Now()
	sched.Start(start)
	nexts := DrainSchedule(sched)
	durations := make([]time.Duration, len(nexts))
	for i, next := range nexts {
		durations[i] = next.Sub(start)
	}
	return durations
}

const drainLimit = 1000000

// DrainSchedule takes all tokens from passed schedule.
// Returns all tokens and finish time.
func DrainSchedule(sched core.Schedule) []time.Time {
	var nexts []time.Time
	for len(nexts) < drainLimit {
		next, ok := sched.Next()
		nexts = append(nexts, next)
		if !ok {
			return nexts
		}
	}
	panic("drain limit reached")
}
