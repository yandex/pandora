// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coretest

import (
	"time"

	"github.com/yandex/pandora/core"
	"github.com/onsi/gomega"
)

func ExpectScheduleNextsStartAt(sched core.Schedule, startAt time.Time, nexts ...time.Duration) {
	beforeStartLeft := sched.Left()
	tokensExpected := len(nexts) - 1 // Last next is finish time.
	gomega.Expect(beforeStartLeft).To(gomega.Equal(tokensExpected))
	sched.Start(startAt)
	actualNexts := DrainScheduleDuration(sched, startAt)
	gomega.Expect(actualNexts).To(gomega.Equal(nexts))
}

func ExpectScheduleNexts(sched core.Schedule, nexts ...time.Duration) {
	ExpectScheduleNextsStartAt(sched, time.Now(), nexts...)
}

const drainLimit = 1000000

// DrainSchedule starts schedule and takes all tokens from it.
// Returns all tokens and finish time relative to start
func DrainScheduleDuration(sched core.Schedule, startAt time.Time) []time.Duration {
	nexts := DrainSchedule(sched)
	durations := make([]time.Duration, len(nexts))
	for i, next := range nexts {
		durations[i] = next.Sub(startAt)
	}
	return durations
}

// DrainSchedule takes all tokens from passed schedule.
// Returns all tokens and finish time.
func DrainSchedule(sched core.Schedule) []time.Time {
	expectedLeft := sched.Left()
	var nexts []time.Time
	for len(nexts) < drainLimit {
		next, ok := sched.Next()
		nexts = append(nexts, next)
		if !ok {
			gomega.Expect(sched.Left()).To(gomega.Equal(0))
			return nexts
		}
		expectedLeft--
		gomega.Expect(sched.Left()).To(gomega.Equal(expectedLeft))
	}
	panic("drain limit reached")
}
