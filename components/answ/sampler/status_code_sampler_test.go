package sampler

import (
	"math"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestFactorSampler(t *testing.T) {
	factor := 10
	fs := newFactorSampler(factor, 600)

	sampleChan := make(chan bool, int(math.Pow(10, 6)))

	var wg sync.WaitGroup

	for range 1000 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 1000 {
				sampleChan <- fs.SampleAnsw([]zapcore.Field{zap.Int(FilterAndSampleGroup, 200)})
			}
		}()
	}

	wg.Wait()
	close(sampleChan)

	samplesExpected, samplesCount := 1, 0
	for sample := range sampleChan {
		samplesCount++

		if sample {
			assert.GreaterOrEqual(t, samplesCount, samplesExpected/100*98) // 2% inaccuracy to take into account data race on writing to sampleChan
			samplesExpected *= factor
		}

		if samplesCount > int(math.Pow(10, 6)) {
			assert.Equal(t, samplesExpected, int(math.Pow(10, 7)))
		}
	}
}

// Код ответа приходит из ответа цели, а бакеты выделены фиксированным числом: 600 у HTTP-пушки,
// 20 у gRPC (guns/http/base.go, guns/grpc/core.go). Значение за границей раньше уходило прямо
// в индекс массива и роняло горутину отстрела, то есть всю стрельбу посередине (LOAD-3696).
func TestFactorSamplerOutOfRangeStatusDoesNotPanic(t *testing.T) {
	fs := newFactorSampler(10, 20)

	for _, code := range []int{-1, 20, 21, 600, 999, 1 << 20} {
		assert.NotPanics(t, func() {
			fs.SampleAnsw([]zapcore.Field{zap.Int(FilterAndSampleGroup, code)})
		}, "код %d не должен ронять стрельбу", code)
	}
}

// Ответ с неизвестным кодом сэмплировать нечем, поэтому он пишется в лог целиком — так же,
// как ответ без группирующего поля вообще.
func TestFactorSamplerOutOfRangeStatusIsSampled(t *testing.T) {
	fs := newFactorSampler(10, 20)

	assert.True(t, fs.SampleAnsw([]zapcore.Field{zap.Int(FilterAndSampleGroup, 999)}))
}

// Границы массива: последний валидный бакет обязан считаться как обычно.
func TestFactorSamplerLastBucketStillCounted(t *testing.T) {
	fs := newFactorSampler(10, 20)

	assert.True(t, fs.SampleAnsw([]zapcore.Field{zap.Int(FilterAndSampleGroup, 19)}), "первый ответ в бакете пишется")
	assert.False(t, fs.SampleAnsw([]zapcore.Field{zap.Int(FilterAndSampleGroup, 19)}), "второй уже сэмплируется")
}
