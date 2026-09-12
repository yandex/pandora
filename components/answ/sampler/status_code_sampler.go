package sampler

import (
	"sync/atomic"

	"github.com/yandex/pandora/components/answ"
	"github.com/yandex/pandora/lib/answlog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const FilterAndSampleGroup = "status"

func NewStatusCodeSampler(pattern any, bucketsCnt int) answlog.LogSampler {
	if pattern == nil {
		return &answlog.PassAllSampler{}
	}

	switch p := pattern.(type) {
	case FactorPattern:
		if p.Factor < 1 {
			zap.L().Info("Incorrect factor for sampling, it must be positive. Disabling sampling")
			return &answlog.DiscardAllSampler{}
		}
		return newFactorSampler(p.Factor, bucketsCnt)
	default:
		zap.L().Info("Unknown sampling pattern. Disabling sampling")
		return &answlog.DiscardAllSampler{}
	}
}

type factorSampler struct {
	counter []*atomic.Int64
	indices []*atomic.Int64
	factor  int64
}

func newFactorSampler(factor int, bucketsCnt int) *factorSampler {
	counter := make([]*atomic.Int64, bucketsCnt)
	indices := make([]*atomic.Int64, bucketsCnt)
	for i := range bucketsCnt {
		counter[i] = new(atomic.Int64)
		indices[i] = new(atomic.Int64)
		indices[i].Add(1)
	}
	return &factorSampler{
		counter: counter,
		indices: indices,
		factor:  int64(factor),
	}
}

func (s *factorSampler) SampleAnsw(fields []zapcore.Field) bool {
	var sample bool

	field, ok := answ.GetField(fields, FilterAndSampleGroup)
	if !ok {
		return true
	}

	statusCode := field.Integer
	// Код приходит из ответа цели, а бакеты выделены фиксированным числом (600 для HTTP-пушки,
	// 20 для gRPC). Нестандартный код вроде 999 раньше давал index out of range прямо в горутине
	// отстрела, то есть ронял всю стрельбу посередине (LOAD-3696). Такой ответ сэмплировать нечем,
	// поэтому пишем его в лог целиком — как и ответ без группирующего поля выше.
	if statusCode < 0 || statusCode >= int64(len(s.counter)) {
		return true
	}

	s.counter[statusCode].Add(1)
	c := s.counter[statusCode].Load()
	i := s.indices[statusCode].Load()
	if c >= i {
		if s.indices[statusCode].CompareAndSwap(i, i*s.factor) {
			sample = true
		}
	}

	return sample
}
