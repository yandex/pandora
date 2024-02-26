// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coretest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core"
)

func ExpectScheduleNextsStartAt(t *testing.T, sched core.Schedule, startAt time.Time, nexts ...time.Duration) {
	beforeStartLeft := sched.Left()
	tokensExpected := len(nexts) - 1 // Last next is finish time.
	require.Equal(t, tokensExpected, beforeStartLeft)
	sched.Start(startAt)
	actualNexts := DrainScheduleDuration(t, sched, startAt)
	require.Equal(t, nexts, actualNexts)
}

func ExpectScheduleNexts(t *testing.T, sched core.Schedule, nexts ...time.Duration) {
	ExpectScheduleNextsStartAt(t, sched, time.Now(), nexts...)
}

func ExpectScheduleNextsStartAtT(t *testing.T, sched core.Schedule, startAt time.Time, nexts ...time.Duration) {
	beforeStartLeft := sched.Left()
	tokensExpected := len(nexts) - 1 // Last next is finish time.
	require.Equal(t, tokensExpected, beforeStartLeft)
	sched.Start(startAt)
	actualNexts := DrainScheduleDuration(t, sched, startAt)
	require.Equal(t, nexts, actualNexts)
}

func ExpectScheduleNextsT(t *testing.T, sched core.Schedule, nexts ...time.Duration) {
	ExpectScheduleNextsStartAtT(t, sched, time.Now(), nexts...)
}

const drainLimit = 1000000

// DrainSchedule starts schedule and takes all tokens from it.
// Returns all tokens and finish time relative to start
func DrainScheduleDuration(t *testing.T, sched core.Schedule, startAt time.Time) []time.Duration {
	nexts := DrainSchedule(t, sched)
	durations := make([]time.Duration, len(nexts))
	for i, next := range nexts {
		durations[i] = next.Sub(startAt)
	}
	return durations
}

// DrainSchedule takes all tokens from passed schedule.
// Returns all tokens and finish time.
func DrainSchedule(t *testing.T, sched core.Schedule) []time.Time {
	expectedLeft := sched.Left()
	var nexts []time.Time
	for len(nexts) < drainLimit {
		next, ok := sched.Next()
		nexts = append(nexts, next)
		if !ok {
			require.Equal(t, 0, sched.Left())
			return nexts
		}
		expectedLeft--
		require.Equal(t, expectedLeft, sched.Left())
	}
	panic("drain limit reached")
}
