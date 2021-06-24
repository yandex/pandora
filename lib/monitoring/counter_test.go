package monitoring

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewCounter(t *testing.T) {

	c := NewCounter("test_counter")

	var initVal int64 = 10

	c.Set(initVal)

	assert.Equal(t, c.Get(), initVal)

	var delta int64 = 10

	c.Add(delta)

	assert.Equal(t, c.Get(), initVal+delta)

	c.Add(-delta)

	assert.Equal(t, c.Get(), initVal)

	str := c.String()

	assert.NotNil(t, str)

	assert.NotEqual(t, "", str)

}

func TestMultiplyParallelAdd(t *testing.T) {

	c := NewCounter("test_counter_parallel")

	maxVal := 1000

	var wg sync.WaitGroup

	for i := 0; i < maxVal; i++ {
		wg.Add(1)
		go func() {
			c.Add(1)
			wg.Done()
		}()
	}

	wg.Wait()

	assert.Equal(t, c.Get(), int64(maxVal))

}

func TestCounterDuplicate(t *testing.T) {
	assert.Panics(t, func() {
		_ = NewCounter("counter")
		_ = NewCounter("counter")
	})

	assert.NotPanics(t, func() {
		_ = NewCounter("counter1")
		_ = NewCounter("counter2")
	})
}
