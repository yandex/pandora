// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.

package coreutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/yandex/pandora/core/schedule"
)

func TestCallbackOnFinishSchedule(t *testing.T) {
	var callbackTimes int
	wrapped := schedule.NewOnce(1)
	testee := NewCallbackOnFinishSchedule(wrapped, func() {
		callbackTimes++
	})
	startAt := time.Now()
	testee.Start(startAt)
	tx, ok := testee.Next()

	require.True(t, ok)
	require.Equal(t, startAt, tx)
	require.Equal(t, 0, callbackTimes)

	tx, ok = testee.Next()
	require.False(t, ok)
	require.Equal(t, startAt, tx)
	require.Equal(t, 1, callbackTimes)

	tx, ok = testee.Next()
	require.False(t, ok)
	require.Equal(t, startAt, tx)
	require.Equal(t, 1, callbackTimes)
}
