// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.

package coreutil

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core/schedule"
)

func TestWaiter_Unstarted(t *testing.T) {
	sched := schedule.NewOnce(1)
	ctx := context.Background()
	w := NewWaiter(sched, ctx)
	var i int
	for ; w.Wait(); i++ {
	}
	require.Equal(t, 1, i)
}

func TestWaiter_WaitAsExpected(t *testing.T) {
	const (
		duration = 100 * time.Millisecond
		ops      = 100
		times    = ops * duration / time.Second
	)
	sched := schedule.NewConst(ops, duration)
	ctx := context.Background()
	w := NewWaiter(sched, ctx)
	start := time.Now()
	sched.Start(start)
	var i int
	for ; w.Wait(); i++ {
	}
	finish := time.Now()

	require.Equal(t, int(times), i)
	dur := finish.Sub(start)
	require.True(t, dur >= duration*(times-1)/times)
	require.True(t, dur < 3*duration) // Smaller interval will be more flaky.
}
func TestWaiter_ContextCanceledBeforeWait(t *testing.T) {
	sched := schedule.NewOnce(1)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	w := NewWaiter(sched, ctx)
	require.False(t, w.Wait())
}

func TestWaiter_ContextCanceledDuringWait(t *testing.T) {
	sched := schedule.NewConstConf(schedule.ConstConfig{Ops: 0.1, Duration: 100 * time.Second})
	timeout := 20 * time.Millisecond
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	w := NewWaiter(sched, ctx)

	require.True(t, w.Wait()) // 0
	require.False(t, w.Wait())

	since := time.Since(start)
	require.True(t, since > timeout)
	require.True(t, since < 10*timeout)
}
