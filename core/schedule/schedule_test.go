package schedule

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coretest"
)

func Test_unlimited(t *testing.T) {
	conf := UnlimitedConfig{50 * time.Millisecond}
	testee := NewUnlimitedConf(conf)
	start := time.Now()
	finish := start.Add(conf.Duration)
	assert.Equal(t, -1, testee.Left())
	testee.Start(start)
	var i int
	for prev := time.Now(); ; i++ {
		left := testee.Left()
		x, ok := testee.Next()
		if !ok {
			break
		}
		assert.Equal(t, -1, left)
		assert.True(t, x.After(prev))
		assert.True(t, x.Before(finish))
	}
	assert.Equal(t, 0, testee.Left())
	assert.Greater(t, i, 50)
}

func TestOnce(t *testing.T) {
	t.Run("started", func(t *testing.T) {
		testee := NewOnce(1)
		coretest.ExpectScheduleNexts(t, testee, 0, 0)
	})

	t.Run("unstarted", func(t *testing.T) {
		testee := NewOnce(1)
		start := time.Now()
		x1, ok := testee.Next()
		threshold := time.Since(start)

		assert.True(t, ok)
		assert.WithinDuration(t, start, x1, threshold)

		x2, ok := testee.Next()
		assert.False(t, ok)
		assert.Equal(t, x1, x2)
	})

}

func TestConst(t *testing.T) {
	tests := []struct {
		name           string
		conf           ConstConfig
		wantN          int64
		wantAssertNext []time.Duration
	}{
		{
			name: "non-zero ops",
			conf: ConstConfig{
				Ops:      1,
				Duration: 2 * time.Second,
			},
			wantN: 2,
			wantAssertNext: []time.Duration{
				0, time.Second, 2 * time.Second,
			},
		},
		{
			name: "zero ops",
			conf: ConstConfig{
				Ops:      0,
				Duration: 2 * time.Second,
			},
			wantN: 0,
			wantAssertNext: []time.Duration{
				2 * time.Second,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			testee := NewConstConf(tt.conf)
			underlying := testee.(*doAtSchedule)

			assert.Equal(t, tt.wantN, underlying.n)
			coretest.ExpectScheduleNexts(t, testee, tt.wantAssertNext...)
		})
	}
}

func TestLine(t *testing.T) {
	tests := []struct {
		name           string
		conf           LineConfig
		wantN          int64
		wantAssertNext []time.Duration
		assert         func(t *testing.T, testee core.Schedule, underlying *doAtSchedule)
	}{
		{
			name: "too small ops",
			conf: LineConfig{
				From:     0,
				To:       1.999,
				Duration: time.Second,
			},
			wantN:          0,
			wantAssertNext: []time.Duration{time.Second},
		},
		{
			name: "const ops",
			conf: LineConfig{
				From:     1,
				To:       1,
				Duration: 2 * time.Second,
			},
			wantN:          2,
			wantAssertNext: []time.Duration{0, time.Second, 2 * time.Second},
		},
		{
			name: "zero start",
			conf: LineConfig{
				From:     0,
				To:       1,
				Duration: 2 * time.Second,
			},
			wantN:          1,
			wantAssertNext: []time.Duration{0, 2 * time.Second},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testee := NewLineConf(tt.conf)
			underlying := testee.(*doAtSchedule)

			assert.Equal(t, tt.wantN, underlying.n)
			coretest.ExpectScheduleNexts(t, testee, tt.wantAssertNext...)
		})
	}
}

func TestLineNonZeroStart(t *testing.T) {
	testee := NewLineConf(LineConfig{
		From:     2,
		To:       8,
		Duration: 2 * time.Second,
	})
	underlying := testee.(*doAtSchedule)

	assert.Equal(t, int64(10), underlying.n)

	start := time.Now()
	testee.Start(start)

	var (
		i  int
		xs []time.Time
		x  time.Time
	)
	for ok := true; ok; i++ {
		x, ok = testee.Next()
		xs = append(xs, x)
	}
	assert.Equal(t, 11, i)
	assert.True(t, sort.SliceIsSorted(xs, func(i, j int) bool {
		return xs[i].Before(xs[j])
	}))
	assert.Equal(t, xs[len(xs)-1], start.Add(2*time.Second))
}

func TestStep(t *testing.T) {
	conf := StepConfig{
		From:     1,
		To:       2,
		Step:     1,
		Duration: 2 * time.Second,
	}
	testee := NewStepConf(conf)
	assert.Equal(t, 6, testee.Left())
}

func TestInstanceStep(t *testing.T) {
	conf := InstanceStepConfig{
		From:         1,
		To:           3,
		Step:         1,
		StepDuration: 2 * time.Second,
	}
	testee := NewInstanceStepConf(conf)
	assert.Equal(t, 3, testee.Left())
}

func BenchmarkLineSchedule(b *testing.B) {
	schedule := NewLine(0, float64(b.N), 2*time.Second)
	benchmarkScheduleNext(b, schedule)
}

func BenchmarkLineScheduleParallel(b *testing.B) {
	schedule := NewLine(0, float64(b.N), 2*time.Second)
	benchmarkScheduleNextParallel(b, schedule)
}

func BenchmarkUnlimitedSchedule(b *testing.B) {
	schedule := NewUnlimited(time.Minute)
	benchmarkScheduleNext(b, schedule)
}

func BenchmarkUnlimitedScheduleParallel(b *testing.B) {
	schedule := NewUnlimited(time.Minute)
	benchmarkScheduleNextParallel(b, schedule)
}

func benchmarkScheduleNextParallel(b *testing.B, schedule core.Schedule) {
	run := func(pb *testing.PB) {
		for pb.Next() {
			schedule.Next()
		}
	}
	schedule.Start(time.Now())
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(run)
}

func benchmarkScheduleNext(b *testing.B, schedule core.Schedule) {
	schedule.Start(time.Now())
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		schedule.Next()
	}
}
