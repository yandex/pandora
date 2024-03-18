package coreutil

import "github.com/yandex/pandora/core"

func ReturnSampleIfBorrowed(s core.Sample) {
	borrowed, ok := s.(core.BorrowedSample)
	if !ok {
		return
	}
	borrowed.Return()
}
