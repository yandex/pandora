package answlog

import "go.uber.org/zap/zapcore"

type LogMasker interface {
	Mask(fields []zapcore.Field) ([]zapcore.Field, error)
}

type PassAllMasker struct{}

func (m *PassAllMasker) Mask(fields []zapcore.Field) ([]zapcore.Field, error) { return fields, nil }
