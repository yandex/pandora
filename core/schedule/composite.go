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
	switch len(scheds) {
	case 0:
		return NewOnce(0)
	case 1:
		return scheds[0]
	}

	var (
		left            = make([]int, len(scheds))
		unknown         bool // If meet any Left() < 0, all previous leftBefore is unknown
		leftAccumulator int  // If unknown, then at least leftBefore accumulated, else exactly leftBefore.
	)
	for i := len(scheds) - 1; i >= 0; i-- {
		left[i] = leftAccumulator
		schedLeft := scheds[i].Left()
		if schedLeft < 0 {
			schedLeft = -1
			unknown = true
			leftAccumulator = -1
		}
		if !unknown {
			leftAccumulator += schedLeft
		}
	}

	return &compositeSchedule{
		scheds:    scheds,
		leftAfter: left,
	}
}

type compositeSchedule struct {
	// Under read lock, goroutine can read slices, it's values, and call values goroutine safe methods.
	// Under write lock, goroutine can do anything.
	rwMu      sync.RWMutex
	scheds    []core.Schedule // At least once schedule. First schedule can be finished.
	leftAfter []int           // Tokens leftBefore, if known exactly, or at least tokens leftBefore otherwise.
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
	// Current schedule is finished, but some are leftBefore.
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
	s.shiftLocked(tx)
	tx, ok = s.scheds[0].Next()
	s.rwMu.Unlock()
	if !ok && schedsLeftNow > 1 {
		// What? Schedule without any tokens? Okay, just retry.
		return s.Next()
	}
	return
}

func (s *compositeSchedule) Left() int {
	s.rwMu.RLock()
	schedsLeft := len(s.scheds)
	leftAfter := int(s.leftAfter[0])
	left := s.scheds[0].Left()
	s.rwMu.RUnlock()
	if schedsLeft == 1 {
		return left
	}
	if left == 0 {
		s.rwMu.Lock()
		currentFinishTime, ok := s.scheds[0].Next()
		if ok {
			panic("current schedule is not finished")
		}
		s.shiftLocked(currentFinishTime)
		s.rwMu.Unlock()
		return s.Left()
	}
	if left < 0 {
		return -1
	}
	return left + leftAfter
}

func (s *compositeSchedule) shiftLocked(currentFinishTime time.Time) {
	s.scheds = s.scheds[1:]
	s.leftAfter = s.leftAfter[1:]
	s.scheds[0].Start(currentFinishTime)
}
