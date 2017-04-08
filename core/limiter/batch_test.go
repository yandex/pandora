package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/pandora/lib/utils"
)

func TestBatchLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	master := newBase(2)
	for i := 0; i < 2; i++ {
		master.control <- struct{}{}
	}
	close(master.control)

	batch := NewBatch(5, master)
	promise := utils.PromiseCtx(ctx, batch.Start)

	i, err := Drain(ctx, batch)
	assert.NoError(t, err)
	// we should take only 5 tick from master
	assert.Equal(t, i, 10)

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestContextCancelInBatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	limitCtx, limitCancel := context.WithCancel(ctx)
	limitCancel()

	master := newBase(10)

	batch := NewBatch(5, master)
	promise := utils.PromiseCtx(limitCtx, batch.Start)
	i, err := Drain(ctx, batch)
	assert.NoError(t, err)
	// we should take only 0 tick from master
	assert.Equal(t, i, 0)

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestContextCancelWhileControlBatchLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	limitCtx, limitCancel := context.WithCancel(ctx)

	master := &base{control: make(chan struct{})}

	batch := NewBatch(5, master)
	promise := utils.PromiseCtx(limitCtx, batch.Start)

	select {
	case master.control <- struct{}{}:
		// we fed master and then cancel context
		limitCancel()
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
