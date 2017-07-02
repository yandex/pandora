package schedule

import (
	"sort"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/yandex/pandora/core"
)

func TestSchedule(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Schedule Suite")
}

var _ = Describe("unlimited", func() {
	It("", func() {
		conf := UnlimitedConfig{50 * time.Millisecond}
		testee := NewUnlimitedConf(conf)
		start := time.Now()
		finish := start.Add(conf.Duration)
		testee.Start(start)
		var i int
		for prev := time.Now(); ; i++ {
			x, ok := testee.Next()
			if !ok {
				break
			}
			Expect(x).To(BeTemporally(">", prev))
			Expect(x).To(BeTemporally("<", finish))
		}
		Expect(i).To(BeNumerically(">", 50))
	})
})

var _ = Describe("once", func() {
	It("started", func() {
		testee := NewOnce(1)
		start := time.Now()
		testee.Start(start)

		x, ok := testee.Next()
		Expect(ok).To(BeTrue())
		Expect(x).To(Equal(start))

		x, ok = testee.Next()
		Expect(ok).To(BeFalse())
		Expect(x).To(Equal(start))
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
		start      time.Time
	)

	JustBeforeEach(func() {
		testee = NewConstConf(conf)
		underlying = testee.(*doAtSchedule)
		start = time.Now()
		testee.Start(start)
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
			x, ok := testee.Next()
			Expect(ok).To(BeTrue())
			Expect(start.Add(time.Second), x)

			x, ok = testee.Next()
			Expect(ok).To(BeTrue())
			Expect(start.Add(2*time.Second), x)

			x, ok = testee.Next()
			Expect(ok).To(BeFalse())
			Expect(start.Add(2*time.Second), x)
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
			x, ok := testee.Next()
			Expect(ok).To(BeFalse())
			Expect(start.Add(2*time.Second), x)
		})
	})
})

var _ = Describe("line", func() {
	var (
		conf       LineConfig
		testee     core.Schedule
		underlying *doAtSchedule
		start      time.Time
	)

	JustBeforeEach(func() {
		testee = NewLineConf(conf)
		underlying = testee.(*doAtSchedule)
		start = time.Now()
		testee.Start(start)
	})

	Context("too small ops", func() {
		BeforeEach(func() {
			conf = LineConfig{
				From:     0,
				To:       1,
				Duration: time.Second,
			}
		})
		It("", func() {
			// Too small ops, so should not do anything.
			Expect(underlying.n).To(BeEquivalentTo(0))

			x, ok := testee.Next()
			Expect(ok).To(BeFalse())
			Expect(start.Add(time.Second), x)
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
			x, ok := testee.Next()
			Expect(ok).To(BeTrue())
			Expect(start.Add(time.Second), x)

			x, ok = testee.Next()
			Expect(ok).To(BeTrue())
			Expect(start.Add(2*time.Second), x)

			x, ok = testee.Next()
			Expect(ok).To(BeFalse())
			Expect(start.Add(2*time.Second), x)
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

			x, ok := testee.Next()
			Expect(ok).To(BeTrue())
			Expect(start.Add(2*time.Second), x)

			x, ok = testee.Next()
			Expect(ok).To(BeFalse())
			Expect(start.Add(2*time.Second), x)
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

			Expect(xs[9]).To(Equal(xs[10]))
			Expect(start.Add(conf.Duration)).To(Equal(xs[10]))
		})
	})

})
