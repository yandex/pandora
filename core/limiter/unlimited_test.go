package limiter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/pandora/lib/utils"
)

func TestContextCancelInUnlimited(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	limitCtx, limitCancel := context.WithCancel(ctx)
	limitCancel()

	unlimited := NewUnlimited()
	promise := utils.PromiseCtx(limitCtx, unlimited.Start)
	_, err := Drain(ctx, unlimited)
	require.NoError(t, err)

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
