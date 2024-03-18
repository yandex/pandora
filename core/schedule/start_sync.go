package schedule

import (
	"sync"

	"go.uber.org/atomic"
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
