// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"time"

	"go.uber.org/atomic"

	"sync"

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
	started   atomic.Bool
	start     time.Time
	startSync sync.Once
	duration  time.Duration
	i         atomic.Int64
	n         int64
	doAt      func(i int64) time.Duration
}

func (s *doAtSchedule) Start(startAt time.Time) {
	if s.started.Swap(true) {
		panic("schedule is already started")
	}
	s.startSync.Do(func() {
		s.start = startAt
	})
}

func (s *doAtSchedule) startOnce(startAt time.Time) {
}

func (s *doAtSchedule) Next() (tx time.Time, ok bool) {
	s.startSync.Do(func() {
		// No allocations here due to benchmark.
		s.started.Store(true)
		s.start = time.Now()
	})
	i := s.i.Inc()
	if i > s.n {
		return s.start.Add(s.duration), false
	}
	return s.start.Add(s.doAt(i)), true
}
