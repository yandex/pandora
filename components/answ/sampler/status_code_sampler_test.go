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
