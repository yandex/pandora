package answlog

import "go.uber.org/zap/zapcore"

type LogFilter interface {
	FilterAnsw([]zapcore.Field) bool
}

type PassAllFilter struct{}

func (f *PassAllFilter) FilterAnsw(fields []zapcore.Field) bool { return true }
