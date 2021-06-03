// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coreutil

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"a.yandex-team.ru/load/projects/pandora/core/schedule"
)

var _ = Describe("waiter", func() {

	It("unstarted", func() {
		sched := schedule.NewOnce(1)
		ctx := context.Background()
		w := NewWaiter(sched, ctx)
		var i int
		for ; w.Wait(); i++ {
		}
		Expect(i).To(BeEquivalentTo(1))
	})

	It("wait as expected", func() {
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
		Expect(i).To(BeEquivalentTo(times))
		Expect(finish.Sub(start)).To(BeNumerically(">=", duration*(times-1)/times))
		Expect(finish.Sub(start)).To(BeNumerically("<", 3*duration)) // Smaller interval will be more flaky.
	})

	It("context canceled before wait", func() {
		sched := schedule.NewOnce(1)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		w := NewWaiter(sched, ctx)
		Expect(w.Wait()).To(BeFalse())
	})

	It("context canceled during wait", func() {
		sched := schedule.NewConstConf(schedule.ConstConfig{Ops: 0.1, Duration: 100 * time.Second})
		timeout := 20 * time.Millisecond
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		w := NewWaiter(sched, ctx)
		Expect(w.Wait()).To(BeTrue()) // 0
		Expect(w.Wait()).To(BeFalse())
		Expect(time.Since(start)).To(BeNumerically(">", timeout))
		Expect(time.Since(start)).To(BeNumerically("<", 10*timeout))
	})

})
