package coreutil

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/yandex/pandora/core/schedule"
)

func TestCoreutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Coreutil Suite")
}

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
		conf := schedule.ConstConfig{100, 100 * time.Millisecond}
		sched := schedule.NewConstConf(conf)
		ctx := context.Background()
		w := NewWaiter(sched, ctx)
		start := time.Now()
		sched.Start(start)
		var i int
		for ; w.Wait(); i++ {
		}
		finish := time.Now()
		Expect(i).To(BeEquivalentTo(10))
		Expect(finish.Sub(start)).To(BeNumerically(">=", conf.Duration))
		Expect(finish.Sub(start)).To(BeNumerically("<", 3*conf.Duration)) // Smaller interval will be more flaky.
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
		timeout := 10 * time.Millisecond
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		w := NewWaiter(sched, ctx)
		Expect(w.Wait()).To(BeFalse())
		Expect(time.Since(start)).To(BeNumerically(">", timeout))
		Expect(time.Since(start)).To(BeNumerically("<", 10*timeout))
	})

})

// Research what method of context done checking is better.
// Approx results for my MacBook Pro 2015 i5:
// <-ctx.Done(), atomic powered.	            5 ns/op parallel, 9 ns/op sequential.
// ctx.Err() != nil, defer and mutex based.     150 ns/op parallel, 60 ns/op sequential.
// Morals:
// Using defer is not blazing fast.
// High concurrency for mutex decreases performance.
// Atomic operations are cool.
func BenchmarkContextDone(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	b.Run("Check Done Parallel", func(b *testing.B) {
		b.RunParallel(func(b *testing.PB) {
			for b.Next() {
				select {
				case <-ctx.Done():
					panic("wtf")
				default:
				}
			}
		})
	})
	b.Run("Check Done", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			select {
			case <-ctx.Done():
				panic("wtf")
			default:
			}
		}
	})
	b.Run("Check Err Parallel", func(b *testing.B) {
		b.RunParallel(func(b *testing.PB) {
			for b.Next() {
				if ctx.Err() != nil {
					panic("wtf")
				}
			}
		})
	})
	b.Run("Check Err", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			if ctx.Err() != nil {
				panic("wtf")
			}
		}
	})
}
