package aggregate

import (
	"log"

	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

type resultListener struct {
	sink chan<- Sample
}

func (rl *resultListener) Sink() chan<- Sample {
	return rl.sink
}

// Implements ResultListener interface
type LoggingResultListener struct {
	resultListener

	source <-chan Sample
}

func (rl *LoggingResultListener) handle(r Sample) {
	log.Println(r)
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
	ch := make(chan Sample, 32)
	return &LoggingResultListener{
		source: ch,
		resultListener: resultListener{
			sink: ch,
		},
	}, nil
}
