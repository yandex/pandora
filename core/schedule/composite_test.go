// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package schedule

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coretest"
	"go.uber.org/atomic"
)

func Test_composite(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		testee := NewComposite()
		coretest.ExpectScheduleNexts(t, testee, 0)
	})

	t.Run("only", func(t *testing.T) {
		testee := NewComposite(NewConst(1, time.Second))
		coretest.ExpectScheduleNexts(t, testee, 0, time.Second)
	})

	t.Run("composite", func(t *testing.T) {
		testee := NewComposite(
			NewConst(1, 2*time.Second),
			NewOnce(2),
			NewConst(0, 5*time.Second),
			NewOnce(0),
			NewOnce(1),
		)
		coretest.ExpectScheduleNexts(t, testee,
			0,
			time.Second,

			2*time.Second,
			2*time.Second,

			7*time.Second,
			7*time.Second, // Finish.
		)
	})

	// Load concurrently, and let race detector do it's work.
	t.Run("race", func(t *testing.T) {
		var (
			nexts          []core.Schedule
			tokensGot      atomic.Int64
			tokensExpected int64
		)
		addOnce := func(v int64) {
			nexts = append(nexts, NewOnce(v))
			tokensExpected += v
		}
		addOnce(100000) // Delay to start concurrent readers.
		for i := 0; i < 100000; i++ {
			// Some work for catching races.
			addOnce(0)
			addOnce(1)
		}
		testee := NewCompositeConf(CompositeConf{nexts})
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for {
					_, ok := testee.Next()
					if !ok {
						return
					}
					tokensGot.Inc()
				}
			}()
		}
		wg.Wait()
		assert.Equal(t, tokensExpected, tokensGot.Load())
	})

	t.Run("left with unknown", func(t *testing.T) {
		unlimitedDuration := time.Second
		testee := NewComposite(
			NewUnlimited(unlimitedDuration),
			NewOnce(0),
			NewConst(1, 2*time.Second),
			NewOnce(1),
		)
		assert.Equal(t, -1, testee.Left())
		startAt := time.Now().Add(-unlimitedDuration)
		testee.Start(startAt)

		unlimitedFinish := startAt.Add(unlimitedDuration)
		sched := testee.(*compositeSchedule).scheds[0]
		assert.Equal(t, unlimitedFinish, sched.(*unlimitedSchedule).finish.Load())

		assert.Equal(t, 3, testee.Left())

		actualNexts := coretest.DrainScheduleDuration(t, testee, unlimitedFinish)
		expectedNests := []time.Duration{
			0,
			time.Second,
			2 * time.Second,
			2 * time.Second, // Finish.
		}
		assert.Equal(t, expectedNests, actualNexts)
	})
}
