// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"go.uber.org/atomic"

	"sync"
)

// StartSync is util to make schedule start goroutine safe.
// See doAtSchedule as example.
type StartSync struct {
	started   atomic.Bool
	startOnce sync.Once
}

func (s *StartSync) IsStarted() bool {
	return s.started.Load()
}

func (s *StartSync) MarkStarted() {
	if s.started.Swap(true) {
		panic("schedule is already started")
	}
}
