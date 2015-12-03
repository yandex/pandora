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

func TestQuadraticRightRoot(t *testing.T) {
	root, err := quadraticRightRoot(1, 1, -6)
	require.NoError(t, err)
	assert.Equal(t, 2.0, root)
}

func TestLinearLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	limiter := NewLinear(5, 6, 1)
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

func TestEmptyLinearLimiterConfig(t *testing.T) {
	lc := &config.Limiter{
		LimiterType: "linear",
		Parameters:  nil,
	}
	l, err := NewLinearFromConfig(lc)

	if err == nil {
		t.Errorf("Should return error if empty config")
	}
	if l != nil {
		t.Errorf("Should return 'nil' if empty config")
	}
}

func TestLinearLimiterFromConfig(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	lc := &config.Limiter{
		LimiterType: "linear",
		Parameters: map[string]interface{}{
			"Period":   0.46,
			"StartRps": 10.0,
			"EndRps":   10.1,
		},
	}
	limiter, err := NewLinearFromConfig(lc)

	if err != nil {
		t.Errorf("Got an error while creating linear limiter: %s", err)
	}
	if limiter == nil {
		t.Errorf("Returned 'nil' with valid config")
	}
	switch tt := limiter.(type) {
	case *linear:
	default:
		t.Errorf("Wrong limiter type returned (expected linear): %T", tt)
	}
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
