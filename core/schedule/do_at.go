// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"time"

	"github.com/yandex/pandora/core"
	"go.uber.org/atomic"
)

// DoAt returns when i'th operation should be performed, assuming that schedule
// started at 0.
type DoAt func(i int64) time.Duration

func NewDoAtSchedule(duration time.Duration, n int64, doAt DoAt) core.Schedule {
	return &doAtSchedule{
		duration: duration,
		n:        n,
		doAt:     doAt,
	}
}

type doAtSchedule struct {
	duration time.Duration
	n        int64
	i        atomic.Int64
	doAt     func(i int64) time.Duration

	StartSync
	start time.Time
}

func (s *doAtSchedule) Start(startAt time.Time) {
	s.MarkStarted()
	s.startOnce.Do(func() {
		s.start = startAt
	})
}

func (s *doAtSchedule) Next() (tx time.Time, ok bool) {
	s.startOnce.Do(func() {
		// No allocations here due to benchmark.
		s.MarkStarted()
		s.start = time.Now()
	})
	i := s.i.Inc() - 1
	if i >= s.n {
		return s.start.Add(s.duration), false
	}
	return s.start.Add(s.doAt(i)), true
}

func (s *doAtSchedule) Left() int {
	left := int(s.n - s.i.Load())
	if left < 0 {
		return 0
	}
	return left
}
