package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yandex/pandora/lib/utils"
)

func TestQuadraticRightRoot(t *testing.T) {
	root, err := quadraticRightRoot(1, 1, -6)
	require.NoError(t, err)
	assert.Equal(t, 2.0, root)
}

func TestLinearLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
	defer cancel()

	limiter := NewLinear(LinearConfig{
		StartRps: 5,
		EndRps:   6,
		Period:   time.Second,
	})
	promise := utils.PromiseCtx(ctx, limiter.Start)

	ch := make(chan int, 100)
	go func() {
		i, err := Drain(ctx, limiter)
		if err != nil {
			t.Fatal(err)
		}
		ch <- i
	}()
	select {

	case i := <-ch:
		assert.Equal(t, 6, i)
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

func TestLinearLimiterFromConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	limiter := NewLinear(LinearConfig{
		Period:   46 * time.Second / 100,
		StartRps: 10.0,
		EndRps:   10.1,
	})

	promise := utils.PromiseCtx(ctx, limiter.Start)

	ch := make(chan int, 100)
	go func() {
		i, err := Drain(ctx, limiter)
		if err != nil {
			t.Fatal(err)
		}
		ch <- i
	}()
	select {
	case i := <-ch:
		assert.Equal(t, 5, i)
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
