// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"time"

	"go.uber.org/atomic"

	"github.com/yandex/pandora/core"
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
	start    time.Time
	duration time.Duration
	i        atomic.Int64
	n        int64
	doAt     func(i int64) time.Duration
}

func (s *doAtSchedule) Start(startAt time.Time) {
	if !s.start.IsZero() {
		panic("schedule is already started")
	}
	s.start = startAt
}

func (s *doAtSchedule) Next() (tx time.Time, ok bool) {
	if s.start.IsZero() {
		s.Start(time.Now())
	}
	i := s.i.Inc()
	if i > s.n {
		return s.start.Add(s.duration), false
	}
	return s.start.Add(s.doAt(i)), true
}
