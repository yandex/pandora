package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yandex/pandora/core"
)

func Test_Num(t *testing.T) {
	var (
		limit int

		p      core.Provider
		ctx    context.Context
		cancel context.CancelFunc
		runRes chan error
	)
	beforeEach := func() {
		limit = 0
		runRes = make(chan error)
	}
	justBeforeEach := func() {
		ctx, cancel = context.WithCancel(context.Background())
		p = NewNumConf(NumConfig{limit})
		go func() {
			runRes <- p.Run(ctx, core.ProviderDeps{})
		}()
	}

	t.Run("unlimited", func(t *testing.T) {
		beforeEach()
		justBeforeEach()

		for i := 0; i < 100; i++ {
			a, ok := p.Acquire()
			assert.True(t, ok)
			assert.Equal(t, i, a)
		}
		cancel()

		res := <-runRes
		assert.NoError(t, res)

		a, ok := p.Acquire()
		assert.False(t, ok)
		assert.Nil(t, a)
	})

	t.Run("unlimited", func(t *testing.T) {
		beforeEach()
		limit = 50
		justBeforeEach()

		for i := 0; i < limit; i++ {
			a, ok := p.Acquire()
			assert.True(t, ok)
			assert.Equal(t, i, a)
		}
		res := <-runRes
		assert.NoError(t, res)

		a, ok := p.Acquire()
		assert.False(t, ok)
		assert.Nil(t, a)
	})
}
