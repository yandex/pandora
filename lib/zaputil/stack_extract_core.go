// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package zaputil

import (
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// NewStackExtractCore returns core that extracts stacktraces from
// error fields with github.com/pkg/errors error types, and append them to
// zapcore.Entry.Stack, on Write.
// That makes stacktraces from errors readable in case of console encoder.
// WARN(skipor): don't call Check of underlying cores, just use LevelEnabler.
// That breaks sampling and other complex logic of choosing log or not entry.
func NewStackExtractCore(c zapcore.Core) zapcore.Core {
	return &errStackExtractCore{c, getBuffer()}
}

type errStackExtractCore struct {
	zapcore.Core
	stacksBuff zapBuffer
}

type stackedErr interface {
	error
	StackTrace() errors.StackTrace
}

type causer interface {
	Cause() error
}

func (c *errStackExtractCore) With(fields []zapcore.Field) zapcore.Core {
	buff := c.cloneBuffer()
	fields = extractFieldsStacksToBuff(buff, fields)
	return &errStackExtractCore{
		c.Core.With(fields),
		buff,
	}
}

func (c *errStackExtractCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	if c.stacksBuff.Len() == 0 && !hasStacksToExtract(fields) {
		return c.Core.Write(ent, fields)
	}
	buff := c.cloneBuffer()
	defer buff.Free()
	fields = extractFieldsStacksToBuff(buff, fields)

	if ent.Stack == "" {
		ent.Stack = buff.String()
	} else {
		// Should be rare case, so allocation is OK.
		ent.Stack = ent.Stack + "\n" + buff.String()
	}
	return c.Core.Write(ent, fields)
}

func (c *errStackExtractCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// HACK(skipor): not calling Check of nested. It's ok while we use simple io/tee cores.
	// But that breaks sampling logic of underlying cores, for example.
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *errStackExtractCore) cloneBuffer() zapBuffer {
	clone := getBuffer()
	_, _ = clone.Write(c.stacksBuff.Bytes())
	return clone
}

func hasStacksToExtract(fields []zapcore.Field) bool {
	for _, field := range fields {
		if field.Type != zapcore.ErrorType {
			continue
		}
		_, ok := field.Interface.(stackedErr)
		if ok {
			return true
		}
	}
	return false
}

func extractFieldsStacksToBuff(buff zapBuffer, fields []zapcore.Field) []zapcore.Field {
	var stacksFound bool
	for i, field := range fields {
		if field.Type != zapcore.ErrorType {
			continue
		}
		stacked, ok := field.Interface.(stackedErr)
		if !ok {
			continue
		}
		if !stacksFound {
			stacksFound = true
			oldFields := fields
			fields = make([]zapcore.Field, len(fields))
			copy(fields, oldFields)
		}
		if cause, ok := stacked.(causer); ok {
			field.Interface = cause.Cause()
		} else {
			field = zap.String(field.Key, stacked.Error())
		}
		fields[i] = field
		appendStack(buff, field.Key, stacked.StackTrace())
	}
	return fields // Cloned in case modifications.
}

func appendStack(buff zapBuffer, key string, stack errors.StackTrace) {
	if buff.Len() != 0 {
		buff.AppendByte('\n')
	}
	buff.AppendString(key)
	buff.AppendString(" stacktrace:")
	stack.Format(zapBufferFmtState{buff}, 'v')
}

type zapBuffer struct{ *buffer.Buffer }

var _ ioStringWriter = zapBuffer{}

type ioStringWriter interface {
	WriteString(s string) (n int, err error)
}

func (b zapBuffer) WriteString(s string) (n int, err error) {
	b.AppendString(s)
	return len(s), nil
}

var bufferPool = buffer.NewPool()

func getBuffer() zapBuffer {
	return zapBuffer{bufferPool.Get()}
}

type zapBufferFmtState struct{ zapBuffer }

var _ fmt.State = zapBufferFmtState{}

func (zapBufferFmtState) Flag(c int) bool {
	switch c {
	case '+':
		return true
	default:
		return false
	}
}

func (zapBufferFmtState) Width() (wid int, ok bool)      { panic("should not be called") }
func (zapBufferFmtState) Precision() (prec int, ok bool) { panic("should not be called") }
