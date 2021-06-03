// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coreutil

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"a.yandex-team.ru/load/projects/pandora/core/schedule"
)

var _ = Describe("callback on finish schedule", func() {
	It("callback once", func() {
		var callbackTimes int
		wrapped := schedule.NewOnce(1)
		testee := NewCallbackOnFinishSchedule(wrapped, func() {
			callbackTimes++
		})
		startAt := time.Now()
		testee.Start(startAt)
		tx, ok := testee.Next()
		Expect(ok).To(BeTrue())
		Expect(tx).To(Equal(startAt))
		Expect(callbackTimes).To(Equal(0))

		tx, ok = testee.Next()
		Expect(ok).To(BeFalse())
		Expect(tx).To(Equal(startAt))
		Expect(callbackTimes).To(Equal(1))

		tx, ok = testee.Next()
		Expect(ok).To(BeFalse())
		Expect(tx).To(Equal(startAt))
		Expect(callbackTimes).To(Equal(1))
	})

})
