// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"sync"
	"time"

	"github.com/yandex/pandora/core"
)

type CompositeConf struct {
	Nested []core.Schedule `config:"nested"`
}

func NewCompositeConf(conf CompositeConf) core.Schedule {
	return NewComposite(conf.Nested...)
}

func NewComposite(scheds ...core.Schedule) core.Schedule {
	if len(scheds) == 0 {
		return NewOnce(0)
	}
	return &compositeSchedule{scheds: scheds}
}

type compositeSchedule struct {
	// Under read lock, goroutine can read schedules and call Next,
	// under write lock, goroutine can start next schedule using previous finish time.
	rwMu   sync.RWMutex
	scheds []core.Schedule // At least once schedule.
}

func (s *compositeSchedule) Start(startAt time.Time) {
	s.rwMu.Lock()
	defer s.rwMu.Unlock()
	s.scheds[0].Start(startAt)
}

func (s *compositeSchedule) Next() (tx time.Time, ok bool) {
	s.rwMu.RLock()
	tx, ok = s.scheds[0].Next()
	if ok {
		s.rwMu.RUnlock()
		return // Got token, all is good.
	}
	schedsLeft := len(s.scheds)
	s.rwMu.RUnlock()
	if schedsLeft == 1 {
		return // All nested schedules has been finished, so composite is finished too.
	}
	// Current schedule is finished, but some are left.
	// Let's start next, with got finish time from previous!
	s.rwMu.Lock()
	schedsLeftNow := len(s.scheds)
	somebodyStartedNextBeforeUs := schedsLeftNow < schedsLeft
	if somebodyStartedNextBeforeUs {
		// Let's just take token.
		tx, ok = s.scheds[0].Next()
		s.rwMu.Unlock()
		if ok || schedsLeftNow == 1 {
			return
		}
		// Very strange. Schedule was started and drained while we was waiting for it.
		// Should very rare, so let's just retry.
		return s.Next()
	}
	s.scheds = s.scheds[1:]
	s.scheds[0].Start(tx)
	tx, ok = s.scheds[0].Next()
	s.rwMu.Unlock()
	if !ok && schedsLeftNow > 1 {
		// What? Schedule without any tokens? Okay, just retry.
		return s.Next()
	}
	return
}
