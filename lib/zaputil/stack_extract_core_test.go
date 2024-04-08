package zaputil

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func noStackFields1() []zapcore.Field {
	return []zapcore.Field{
		zap.String("0", "0"), zap.Error(fmt.Errorf("fields 1")),
	}
}

func noStackFields2() []zapcore.Field {
	return []zapcore.Field{
		zap.String("1", "1"), zap.Error(fmt.Errorf("fields 2")),
	}
}

func Test_StackExtractCore(t *testing.T) {
	t.Run("check integration", func(t *testing.T) {
		nested, logs := observer.New(zap.DebugLevel)
		log := zap.New(NewStackExtractCore(nested))

		log.Debug("test", noStackFields1()...)
		assert.Equal(t, 1, logs.Len())
		entry := logs.All()[0]
		assert.Equal(t, "test", entry.Message)
		assert.Equal(t, noStackFields1(), entry.Context)
	})

	t.Run("no stacks", func(t *testing.T) {
		nested, logs := observer.New(zap.DebugLevel)
		testee := NewStackExtractCore(nested)

		testee = testee.With(noStackFields1())
		entry := zapcore.Entry{Message: "test"}
		_ = testee.Write(entry, noStackFields2())

		assert.Equal(t, 1, logs.Len())
		assert.Equal(
			t,
			observer.LoggedEntry{Entry: entry, Context: append(noStackFields1(), noStackFields2()...)},
			logs.All()[0],
		)
	})

	t.Run("stack in write", func(t *testing.T) {
		const sampleErrMsg = "stacked error msg"
		sampleErr := errors.New(sampleErrMsg)
		sampleStack := fmt.Sprintf("%+v", sampleErr.(stackedErr).StackTrace())

		nested, logs := observer.New(zap.DebugLevel)
		testee := NewStackExtractCore(nested)

		fields := append(noStackFields1(), zap.Error(sampleErr))
		fieldsCopy := make([]zapcore.Field, len(fields))
		copy(fieldsCopy, fields)
		entry := zapcore.Entry{Message: "test"}
		_ = testee.Write(entry, fields)

		expectedEntry := entry
		expectedEntry.Stack = "error stacktrace:" + sampleStack
		assert.Equal(t, 1, logs.Len())
		assert.Equal(
			t,
			observer.LoggedEntry{
				Entry:   expectedEntry,
				Context: append(noStackFields1(), zap.String("error", sampleErrMsg)),
			},
			logs.All()[0],
		)
		assert.Equal(t, fieldsCopy, fields)
	})

	t.Run("stack in with", func(t *testing.T) {
		const sampleErrMsg = "stacked error msg"
		sampleCause := fmt.Errorf(sampleErrMsg)
		sampleErr := errors.WithStack(sampleCause)
		sampleStack := fmt.Sprintf("%+v", sampleErr.(stackedErr).StackTrace())

		nested, logs := observer.New(zap.DebugLevel)
		testee := NewStackExtractCore(nested)

		fields := append(noStackFields1(), zap.Error(sampleErr))
		fieldsCopy := make([]zapcore.Field, len(fields))
		copy(fieldsCopy, fields)
		entry := zapcore.Entry{Message: "test"}
		testee = testee.With(fields)
		_ = testee.Write(entry, nil)

		expectedEntry := entry
		expectedEntry.Stack = "error stacktrace:" + sampleStack
		assert.Equal(t, 1, logs.Len())
		assert.Equal(
			t,
			observer.LoggedEntry{
				Entry:   expectedEntry,
				Context: append(noStackFields1(), zap.Error(sampleCause)),
			},
			logs.All()[0],
		)
		assert.Equal(t, fieldsCopy, fields)
	})

	t.Run("stacks join", func(t *testing.T) {
		const sampleErrMsg = "stacked error msg"
		sampleErr := errors.New(sampleErrMsg)
		sampleStack := fmt.Sprintf("%+v", sampleErr.(stackedErr).StackTrace())

		nested, logs := observer.New(zap.DebugLevel)
		testee := NewStackExtractCore(nested)

		const entryStack = "entry stack"
		entry := zapcore.Entry{Message: "test", Stack: entryStack}
		const customKey = "custom-key"
		_ = testee.Write(entry, []zapcore.Field{zap.NamedError(customKey, sampleErr)})

		expectedEntry := entry
		expectedEntry.Stack = entryStack + "\n" + customKey + " stacktrace:" + sampleStack
		assert.Equal(t, 1, logs.Len())

		assert.Equal(
			t,
			observer.LoggedEntry{
				Entry:   expectedEntry,
				Context: []zapcore.Field{zap.String(customKey, sampleErrMsg)},
			},
			logs.All()[0],
		)
	})
}
