// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package aggregator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	coremock "github.com/yandex/pandora/core/mocks"
	"github.com/yandex/pandora/lib/testutil"
)

func TestReporter_DroppedErr(t *testing.T) {
	core, entries := observer.New(zap.DebugLevel)
	zap.ReplaceGlobals(zap.New(core))
	defer testutil.ReplaceGlobalLogger()
	reporter := NewReporter(ReporterConfig{1})
	reporter.Report(1)

	assert.Nil(t, reporter.DroppedErr())
	reporter.Report(2)
	err := reporter.DroppedErr()
	require.Error(t, err)

	assert.EqualValues(t, 1, err.(*SomeSamplesDropped).Dropped)
	assert.Equal(t, 1, entries.Len())
}

func TestReporter_BorrowedSampleReturnedOnDrop(t *testing.T) {
	reporter := NewReporter(ReporterConfig{1})

	reporter.Report(1)
	borrowed := &coremock.BorrowedSample{}
	borrowed.On("Return").Once()

	reporter.Report(borrowed)
	borrowed.AssertExpectations(t)
}
