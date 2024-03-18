package schedule

import (
	"time"

	"github.com/yandex/pandora/core"
)

// NewOnce returns schedule that emits all passed operation token at start time.
// That is, is schedule for zero duration, unlimited RPS, and n operations.
func NewOnce(n int64) core.Schedule {
	return NewDoAtSchedule(0, n, func(i int64) time.Duration {
		return 0
	})
}

type OnceConfig struct {
	Times int64 `validate:"min=1"` // N is decoded like bool
}

func NewOnceConf(conf OnceConfig) core.Schedule {
	return NewOnce(conf.Times)
}
