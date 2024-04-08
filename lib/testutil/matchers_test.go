package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlakyPassed(t *testing.T) {
	var run int
	RunFlaky(t, func(t TestingT) {
		run++
		assert.True(t, run >= 3)
	})
}

func TestFlakyPanic(t *testing.T) {
	var run int
	RunFlaky(t, func(t TestingT) {
		run++
		require.True(t, run >= 3)
	})
}
