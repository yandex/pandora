package aggregator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coremock "github.com/yandex/pandora/core/mocks"
	"github.com/yandex/pandora/lib/testutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestReporter_DroppedErr(t *testing.T) {
	core, entries := observer.New(zap.DebugLevel)
	zap.ReplaceGlobals(zap.New(core))
	defer testutil.ReplaceGlobalLogger()
	reporter := NewReporter(ReporterConfig{1})
	reporter.Report(1)

	assert.NoError(t, reporter.DroppedErr())
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
