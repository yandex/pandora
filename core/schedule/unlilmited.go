// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"time"

	"github.com/yandex/pandora/core"
)

// NewUnlimited returns schedule that generates unlimited ops for passed duration.
func NewUnlimited(duration time.Duration) core.Schedule {
	return &unlimitedSchedule{duration: duration}
}

type UnlimitedConfig struct {
	Duration time.Duration `validate:"min-time=1ms"`
}

func NewUnlimitedConf(conf UnlimitedConfig) core.Schedule {
	return NewUnlimited(conf.Duration)
}

type unlimitedSchedule struct {
	finish   time.Time
	duration time.Duration
}

func (s *unlimitedSchedule) Start(startAt time.Time) {
	if !s.finish.IsZero() {
		panic("schedule is already started")
	}
	s.finish = startAt.Add(s.duration)
}

func (s *unlimitedSchedule) Next() (tx time.Time, ok bool) {
	if s.finish.IsZero() {
		s.Start(time.Now())
	}
	now := time.Now()
	if now.Before(s.finish) {
		return now, true
	}
	return s.finish, false
}
