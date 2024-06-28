package coreutil

import (
	"context"
	"time"

	"github.com/yandex/pandora/core"
)

const MaxOverdueDuration = 2 * time.Second

// Waiter goroutine unsafe wrapper for efficient waiting schedule.
type Waiter struct {
	sched           core.Schedule
	overdueDuration time.Duration

	// Lazy initialized.
	timer   *time.Timer
	lastNow time.Time
}

func NewWaiter(sched core.Schedule) *Waiter {
	return &Waiter{sched: sched}
}

// Wait waits for next waiter schedule event.
// Returns true, if event successfully waited, or false
// if waiter context is done, or schedule finished.
func (w *Waiter) Wait(ctx context.Context) (ok bool) {
	// Check, that context is not done. Very quick: 5 ns for op, due to benchmark.
	select {
	case <-ctx.Done():
		w.overdueDuration = 0
		return false
	default:
	}
	next, ok := w.sched.Next()
	if !ok {
		w.overdueDuration = 0
		return false
	}
	// Get current time lazily.
	// For once schedule, for example, we need to get it only once.
	waitFor := next.Sub(w.lastNow)
	if waitFor <= 0 {
		w.overdueDuration = 0 - waitFor
		return true
	}
	w.lastNow = time.Now()
	waitFor = next.Sub(w.lastNow)
	if waitFor <= 0 {
		w.overdueDuration = 0 - waitFor
		return true
	}
	w.overdueDuration = 0
	// Lazy init. We don't need timer for unlimited and once schedule.
	if w.timer == nil {
		w.timer = time.NewTimer(waitFor)
	} else {
		w.timer.Reset(waitFor)
	}
	select {
	case <-w.timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

// IsSlowDown returns true, if schedule contains 2 elements before current time.
func (w *Waiter) IsSlowDown(ctx context.Context) (ok bool) {
	select {
	case <-ctx.Done():
		return false
	default:
		return w.overdueDuration >= MaxOverdueDuration
	}
}

// IsFinished is quick check, that wait context is not canceled and there are some tokens left in
// schedule.
func (w *Waiter) IsFinished(ctx context.Context) (ok bool) {
	select {
	case <-ctx.Done():
		return true
	default:
		return w.sched.Left() == 0
	}
}
