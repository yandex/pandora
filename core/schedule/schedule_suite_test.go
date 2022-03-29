package schedule

import (
	"sort"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coretest"
	"github.com/yandex/pandora/lib/ginkgoutil"
)

func TestSchedule(t *testing.T) {
	ginkgoutil.RunSuite(t, "Schedule Suite")
}

var _ = Describe("unlimited", func() {
	It("", func() {
		conf := UnlimitedConfig{50 * time.Millisecond}
		testee := NewUnlimitedConf(conf)
		start := time.Now()
		finish := start.Add(conf.Duration)
		Expect(testee.Left()).To(BeEquivalentTo(-1))
		testee.Start(start)
		var i int
		for prev := time.Now(); ; i++ {
			left := testee.Left()
			x, ok := testee.Next()
			if !ok {
				break
			}
			Expect(left).To(BeEquivalentTo(-1))
			Expect(x).To(BeTemporally(">", prev))
			Expect(x).To(BeTemporally("<", finish))
		}
		Expect(testee.Left()).To(BeEquivalentTo(0))
		Expect(i).To(BeNumerically(">", 50))
	})
})

var _ = Describe("once", func() {
	It("started", func() {
		testee := NewOnce(1)
		coretest.ExpectScheduleNexts(testee, 0, 0)
	})

	It("unstarted", func() {
		testee := NewOnce(1)
		start := time.Now()
		x1, ok := testee.Next()
		threshold := time.Since(start)

		Expect(ok).To(BeTrue())
		Expect(x1).To(BeTemporally("~", start, threshold))

		x2, ok := testee.Next()
		Expect(ok).To(BeFalse())
		Expect(x2).To(Equal(x1))
	})

})

var _ = Describe("const", func() {
	var (
		conf       ConstConfig
		testee     core.Schedule
		underlying *doAtSchedule
	)

	JustBeforeEach(func() {
		testee = NewConstConf(conf)
		underlying = testee.(*doAtSchedule)
	})

	Context("non-zero ops", func() {
		BeforeEach(func() {
			conf = ConstConfig{
				Ops:      1,
				Duration: 2 * time.Second,
			}
		})
		It("", func() {
			Expect(underlying.n).To(BeEquivalentTo(2))
			coretest.ExpectScheduleNexts(testee, 0, time.Second, 2*time.Second)
		})
	})

	Context("zero ops", func() {
		BeforeEach(func() {
			conf = ConstConfig{
				Ops:      0,
				Duration: 2 * time.Second,
			}
		})
		It("", func() {
			Expect(underlying.n).To(BeEquivalentTo(0))
			coretest.ExpectScheduleNexts(testee, 2*time.Second)
		})
	})
})

var _ = Describe("line", func() {
	var (
		conf       LineConfig
		testee     core.Schedule
		underlying *doAtSchedule
	)

	JustBeforeEach(func() {
		testee = NewLineConf(conf)
		underlying = testee.(*doAtSchedule)
	})

	Context("too small ops", func() {
		BeforeEach(func() {
			conf = LineConfig{
				From:     0,
				To:       1.999,
				Duration: time.Second,
			}
		})
		It("", func() {
			// Too small ops, so should not do anything.
			Expect(underlying.n).To(BeEquivalentTo(0))
			coretest.ExpectScheduleNexts(testee, time.Second)
		})
	})

	Context("const ops", func() {
		BeforeEach(func() {
			conf = LineConfig{
				From:     1,
				To:       1,
				Duration: 2 * time.Second,
			}
		})

		It("", func() {
			Expect(underlying.n).To(BeEquivalentTo(2))
			coretest.ExpectScheduleNexts(testee, 0, time.Second, 2*time.Second)
		})
	})

	Context("zero start", func() {
		BeforeEach(func() {
			conf = LineConfig{
				From:     0,
				To:       1,
				Duration: 2 * time.Second,
			}
		})

		It("", func() {
			Expect(underlying.n).To(BeEquivalentTo(1))
			coretest.ExpectScheduleNexts(testee, 0, 2*time.Second)
		})
	})

	Context("non zero start", func() {
		BeforeEach(func() {
			conf = LineConfig{
				From:     2,
				To:       8,
				Duration: 2 * time.Second,
			}
		})

		It("", func() {
			Expect(underlying.n).To(BeEquivalentTo(10))
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
			Expect(i).To(Equal(11))
			Expect(sort.SliceIsSorted(xs, func(i, j int) bool {
				return xs[i].Before(xs[j])
			})).To(BeTrue())
			Expect(start.Add(conf.Duration)).To(Equal(xs[len(xs)-1]))
		})
	})

})

var _ = Describe("step", func() {
	It("", func() {
		conf := StepConfig{
			From:     1,
			To:       2,
			Step:     1,
			Duration: 2 * time.Second,
		}
		testee := NewStepConf(conf)
		Expect(testee.Left()).To(Equal(6))

	})

})

var _ = Describe("instance_step", func() {
	It("", func() {
		conf := InstanceStepConfig{
			From:         1,
			To:           3,
			Step:         1,
			StepDuration: 2 * time.Second,
		}
		testee := NewInstanceStepConf(conf)
		Expect(testee.Left()).To(Equal(3))

	})

})

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
