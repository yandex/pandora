package errutil

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/yandex/pandora/lib/ginkgoutil"
)

func TestErrutil(t *testing.T) {
	ginkgoutil.RunSuite(t, "Errutil Suite")
}

var _ = Describe("Iscoreutil.IsCtxErroror", func() {
	canceledContext, cancel := context.WithCancel(context.Background())
	cancel()

	It("nil error", func() {
		Expect(IsCtxError(context.Background(), nil)).To(BeTrue())
		Expect(IsCtxError(canceledContext, nil)).To(BeTrue())
	})

	It("context error", func() {
		Expect(IsCtxError(context.Background(), context.Canceled)).To(BeFalse())
		Expect(IsCtxError(canceledContext, context.Canceled)).To(BeTrue())
	})

	It("caused by context error", func() {
		Expect(IsCtxError(context.Background(), errors.Wrap(context.Canceled, "new err"))).To(BeFalse())
		Expect(IsCtxError(canceledContext, errors.Wrap(context.Canceled, "new err"))).To(BeTrue())
	})

	It("default error wrapping has defferent result", func() {
		Expect(IsCtxError(context.Background(), fmt.Errorf("new err %w", context.Canceled))).To(BeFalse())
		Expect(IsCtxError(canceledContext, fmt.Errorf("new err %w", context.Canceled))).To(BeFalse())
	})

	It("usual error", func() {
		err := errors.New("new err")
		Expect(IsCtxError(canceledContext, err)).To(BeFalse())
		Expect(IsCtxError(context.Background(), err)).To(BeFalse())
	})
})
