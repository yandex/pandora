package aggregate

import (
	"fmt"
	"log"

	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

// Implements ResultListener interface
type LoggingResultListener struct {
	resultListener

	source <-chan interface{}
}

func (rl *LoggingResultListener) handle(r interface{}) {
	r, ok := r.(fmt.Stringer)
	if !ok {
		log.Println("Can't convert result to String")
	} else {
		log.Println(r)
	}
}

func (rl *LoggingResultListener) Start(ctx context.Context) error {
loop:
	for {
		select {
		case r := <-rl.source:
			rl.handle(r)
		case <-ctx.Done():
			// Context is done, but we should read all data from source
			for {
				select {
				case r := <-rl.source:
					rl.handle(r)
				default:
					break loop
				}
			}
		}
	}
	return nil
}

func NewLoggingResultListener(*config.ResultListener) (ResultListener, error) {
	ch := make(chan interface{}, 32)
	return &LoggingResultListener{
		source: ch,
		resultListener: resultListener{
			sink: ch,
		},
	}, nil
}
