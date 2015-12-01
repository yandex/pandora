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

func TestContextCancelInUnlimited(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	limitCtx, limitCancel := context.WithCancel(ctx)
	limitCancel()

	lc := &config.Limiter{
		LimiterType: "unlimited",
		Parameters:  nil,
	}

	unlimited, err := NewUnlimitedFromConfig(lc)
	assert.NoError(t, err)
	promise := utils.PromiseCtx(limitCtx, unlimited.Start)
	_, err = Drain(ctx, unlimited)
	assert.NoError(t, err)

	select {
	case err := <-promise:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}
