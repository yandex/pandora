package limiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/yandex/pandora/utils"
)

func TestSizeLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	master := &limiter{control: make(chan struct{}, 10)}
	for i := 0; i < 10; i++ {
		master.control <- struct{}{}
	}

	size := NewSize(5, master)
	promise := utils.PromiseCtx(ctx, size.Start)

	i, err := Drain(ctx, size)
	assert.NoError(t, err)
	// we should take only 5 tick from master
	assert.Equal(t, i, 5)

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestEmptySizeLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	master := &limiter{control: make(chan struct{}, 10)}
	close(master.control)

	size := NewSize(5, master)
	promise := utils.PromiseCtx(ctx, size.Start)

	i, err := Drain(ctx, size)
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

func TestContextCancelInSizeLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sizeCtx, sizeCancel := context.WithCancel(ctx)
	sizeCancel()

	master := &limiter{control: make(chan struct{}, 10)}

	size := NewSize(5, master)
	promise := utils.PromiseCtx(sizeCtx, size.Start)
	i, err := Drain(ctx, size)
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

func TestContextCancelWhileControlSizeLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	sizeCtx, sizeCancel := context.WithCancel(ctx)

	master := &limiter{control: make(chan struct{})}

	size := NewSize(5, master)
	promise := utils.PromiseCtx(sizeCtx, size.Start)

	select {
	case master.control <- struct{}{}:
		// we fed master and then cancel context
		sizeCancel()
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
