// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package zaputil

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
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

var _ = Describe("stack_extract_core", func() {

	It("check integration", func() {
		nested, logs := observer.New(zap.DebugLevel)
		log := zap.New(NewStackExtractCore(nested))

		log.Debug("test", noStackFields1()...)
		Expect(logs.Len()).To(Equal(1))
		entry := logs.All()[0]
		Expect(entry.Message).To(Equal("test"))
		Expect(entry.Context).To(Equal(noStackFields1()))
	})

	It("no stacks", func() {
		nested, logs := observer.New(zap.DebugLevel)
		testee := NewStackExtractCore(nested)

		testee = testee.With(noStackFields1())
		entry := zapcore.Entry{Message: "test"}
		_ = testee.Write(entry, noStackFields2())

		Expect(logs.Len()).To(Equal(1))
		Expect(logs.All()[0]).To(Equal(
			observer.LoggedEntry{Entry: entry, Context: append(noStackFields1(), noStackFields2()...)},
		))
	})

	It("stack in write", func() {
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
		Expect(logs.Len()).To(Equal(1))
		Expect(logs.All()[0]).To(Equal(
			observer.LoggedEntry{
				Entry:   expectedEntry,
				Context: append(noStackFields1(), zap.String("error", sampleErrMsg)),
			},
		))
		Expect(fields).To(Equal(fieldsCopy))
	})

	It("stack in with", func() {
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
		Expect(logs.Len()).To(Equal(1))
		Expect(logs.All()[0]).To(Equal(
			observer.LoggedEntry{
				Entry:   expectedEntry,
				Context: append(noStackFields1(), zap.Error(sampleCause)),
			},
		))
		Expect(fields).To(Equal(fieldsCopy))
	})

	It("stacks join", func() {
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
		Expect(logs.Len()).To(Equal(1))
		Expect(logs.All()[0]).To(Equal(
			observer.LoggedEntry{
				Entry:   expectedEntry,
				Context: []zapcore.Field{zap.String(customKey, sampleErrMsg)},
			},
		))
	})

})
