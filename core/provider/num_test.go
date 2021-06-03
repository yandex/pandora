package provider

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"a.yandex-team.ru/load/projects/pandora/core"
)

var _ = Describe("Num", func() {
	var (
		limit int

		p      core.Provider
		ctx    context.Context
		cancel context.CancelFunc
		runRes chan error
	)
	BeforeEach(func() {
		limit = 0
		runRes = make(chan error)
	})
	JustBeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		p = NewNumConf(NumConfig{limit})
		go func() {
			runRes <- p.Run(ctx, core.ProviderDeps{})
		}()
	})

	It("unlimited", func() {
		for i := 0; i < 100; i++ {
			a, ok := p.Acquire()
			Expect(ok).To(BeTrue())
			Expect(a).To(Equal(i))
		}
		cancel()
		Expect(<-runRes).To(BeNil())
		a, ok := p.Acquire()
		Expect(ok).To(BeFalse())
		Expect(a).To(BeNil())
	}, 1)

	Context("unlimited", func() {
		BeforeEach(func() {
			limit = 50
		})
		It("", func() {
			for i := 0; i < limit; i++ {
				a, ok := p.Acquire()
				Expect(ok).To(BeTrue())
				Expect(a).To(Equal(i))
			}
			a, ok := p.Acquire()
			Expect(ok).To(BeFalse())
			Expect(a).To(BeNil())
			Expect(<-runRes).To(BeNil())
		}, 1)

	})

})
