package limiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/yandex/pandora/config"
	"github.com/yandex/pandora/utils"
)

func TestPeriodicLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	limiterCtx, cancelLimiter := context.WithCancel(ctx)

	limiter := NewPeriodic(time.Millisecond * 2)
	promise := utils.PromiseCtx(limiterCtx, limiter.Start)

	ch := make(chan int)
	go func() {
		i, err := Drain(ctx, limiter)
		if err != nil {
			t.Fatal(err)
		}
		ch <- i
	}()
	time.Sleep(time.Millisecond * 7)
	cancelLimiter()
	select {

	case i := <-ch:
		// we should take only 4 ticks from ticker (1 in the beginning and 3 after 6 milliseconds)
		assert.Equal(t, 4, i)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestEmptyPeriodicLimiterConfig(t *testing.T) {
	lc := &config.Limiter{
		LimiterType: "periodic",
		Parameters:  nil,
	}
	l, err := NewPeriodicFromConfig(lc)

	if err == nil {
		t.Errorf("Should return error if empty config")
	}
	if l != nil {
		t.Errorf("Should return 'nil' if empty config")
	}
}

func TestPeriodicLimiterNoBatch(t *testing.T) {
	lc := &config.Limiter{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period": 0.46,
		},
	}
	l, err := NewPeriodicFromConfig(lc)

	if err != nil {
		t.Errorf("Got an error while creating periodic limiter: %s", err)
	}
	if l == nil {
		t.Errorf("Returned 'nil' with valid config")
	}
	switch tt := l.(type) {
	case *periodic:
	default:
		t.Errorf("Wrong limiter type returned (expected periodicLimiter): %T", tt)
	}
}

func TestPeriodicLimiterBatch(t *testing.T) {
	lc := &config.Limiter{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    0.46,
			"BatchSize": 3.0,
		},
	}
	l, err := NewPeriodicFromConfig(lc)

	if err != nil {
		t.Errorf("Got an error while creating periodic limiter: %s", err)
	}
	if l == nil {
		t.Errorf("Returned 'nil' with valid config")
	}
	switch tt := l.(type) {
	case *batch:
	default:
		t.Errorf("Wrong limiter type returned (expected batchLimiter): %T", tt)
	}
}

func TestPeriodicLimiterBatchMaxCount(t *testing.T) {
	lc := &config.Limiter{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    0.46,
			"BatchSize": 3.0,
			"MaxCount":  5.0,
		},
	}
	l, err := NewPeriodicFromConfig(lc)

	if err != nil {
		t.Errorf("Got an error while creating periodic limiter: %s", err)
	}
	if l == nil {
		t.Errorf("Returned 'nil' with valid config")
	}
	switch tt := l.(type) {
	case *batch:
	default:
		t.Errorf("Wrong limiter type returned (expected batchLimiter): %T", tt)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	promise := utils.PromiseCtx(ctx, l.Start)
	i, err := Drain(ctx, l)
	assert.NoError(t, err)
	// we should take only 0 tick from master
	assert.Equal(t, i, 15)

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
