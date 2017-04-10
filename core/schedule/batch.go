package schedule

import (
	"context"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/utils"
)

// batch implements core.Schedule interface
type batch struct {
	base
	master    core.Schedule
	batchSize int
}

var _ core.Schedule = (*batch)(nil)

func (bl *batch) Start(ctx context.Context) error {
	defer close(bl.control)
	masterCtx, cancelMaster := context.WithCancel(ctx)
	masterPromise := utils.PromiseCtx(masterCtx, bl.master.Start)
loop:
	for {
		select {
		case _, more := <-bl.master.Control():
			if !more {
				break loop
			}
			for i := 0; i < bl.batchSize; i++ {
				select {
				case bl.control <- struct{}{}:
				case <-ctx.Done():
					break loop
				}
			}
		case <-ctx.Done():
			break loop
		}
	}
	cancelMaster()
	err := <-masterPromise
	return err
}

// NewBatch returns batch limiter that makes batch with size batchSize on every master tick
// master shouldn't be started
func NewBatch(batchSize int, master core.Schedule) (l core.Schedule) {
	return &batch{
		base:      base{make(chan struct{})},
		master:    master,
		batchSize: batchSize,
	}
}
