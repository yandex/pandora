package limiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"

	"github.com/yandex/pandora/utils"
)

func TestQuadraticRightRoot(t *testing.T) {
	root, err := quadraticRightRoot(1, 1, -6)
	require.NoError(t, err)
	assert.Equal(t, 2, root)
}

func TestLinearLimiter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	defer cancel()

	limiterCtx, cancelLimiter := context.WithCancel(ctx)

	limiter := NewLinear(5, 6, 1*time.Second)
	promise := utils.PromiseCtx(limiterCtx, limiter.Start)

	ch := make(chan int, 100)
	go func() {
		i, err := Drain(ctx, limiter)
		if err != nil {
			t.Fatal(err)
		}
		ch <- i
	}()
	time.Sleep(time.Second * 3)
	cancelLimiter()
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
