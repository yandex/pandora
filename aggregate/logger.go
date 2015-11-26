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

func (rl *LoggingResultListener) Start(ctx context.Context) error {
loop:
	for {
		select {
		case r := <-rl.source:
			log.Println(r)
		case <-ctx.Done():
			break loop
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
