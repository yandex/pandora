package answlog

import "go.uber.org/zap/zapcore"

const FilterAndSampleGroup = "status"

type LogSampler interface {
	SampleAnsw([]zapcore.Field) bool
}

type PassAllSampler struct{}

func (s *PassAllSampler) SampleAnsw(fields []zapcore.Field) bool { return true }

type DiscardAllSampler struct{}

func (s *DiscardAllSampler) SampleAnsw(fields []zapcore.Field) bool { return false }
