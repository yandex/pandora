package monitoring

import (
	"sync"
	"testing"
)

func TestInstanceTrackerCountsAndPeak(t *testing.T) {
	var tr InstanceTracker

	tr.OnStart(1)
	tr.OnStart(2)
	if got := tr.String(); got != "2" {
		t.Fatalf("занятых: got %s, want 2", got)
	}
	tr.OnFinish(1)
	tr.OnFinish(2)
	if got := tr.String(); got != "0" {
		t.Fatalf("после финиша: got %s, want 0", got)
	}
	// Flush отдаёт пик с прошлого Flush и сбрасывает его в текущее значение.
	if peak := tr.Flush(); peak != 2 {
		t.Fatalf("пик: got %d, want 2", peak)
	}
	if peak := tr.Flush(); peak != 0 {
		t.Fatalf("пик после сброса: got %d, want 0", peak)
	}
}

func TestInstanceTrackerParallel(t *testing.T) {
	var tr InstanceTracker
	const goroutines, iterations = 64, 1000

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				tr.OnStart(id)
				tr.OnFinish(id)
			}
		}(i)
	}
	wg.Wait()

	if got := tr.String(); got != "0" {
		t.Fatalf("после парных вызовов занятых должно быть 0, got %s", got)
	}
	if peak := tr.Flush(); peak < 1 || peak > goroutines {
		t.Fatalf("пик вне диапазона 1..%d: got %d", goroutines, peak)
	}
}
