package limiter

import (
	"github.com/yandex/pandora/utils"
	"golang.org/x/net/context"
)

// batch implements Limiter interface
type batch struct {
	limiter
	master    Limiter
	batchSize int
}

var _ Limiter = (*batch)(nil)

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
				case bl.control <- true:
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
func NewBatch(batchSize int, master Limiter) (l Limiter) {
	return &batch{
		limiter:   limiter{make(chan bool)},
		master:    master,
		batchSize: batchSize,
	}
}
