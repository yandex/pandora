package coreutil

import (
	"context"
	"testing"

	"github.com/yandex/pandora/lib/testutil"
)

func TestCoreutil(t *testing.T) {
	testutil.RunSuite(t, "Coreutil Suite")
}

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
