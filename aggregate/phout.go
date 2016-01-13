package aggregate

import (
	"bufio"
	"os"
	"time"

	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

type PhoutResultListener struct {
	resultListener

	source <-chan *Sample
	phout  *bufio.Writer
	buffer []byte
}

func (rl *PhoutResultListener) handle(s *Sample) error {

	rl.buffer = s.AppendToPhout(rl.buffer)
	_, err := rl.phout.Write(rl.buffer)
	rl.buffer = rl.buffer[:0]
	ReleaseSample(s)
	return err
}

func (rl *PhoutResultListener) Start(ctx context.Context) error {
	defer rl.phout.Flush()
	shouldFlush := time.NewTicker(1 * time.Second).C
loop:
	for {
		select {
		case r := <-rl.source:
			if err := rl.handle(r); err != nil {
				return err
			}
			select {
			case <-shouldFlush:
				rl.phout.Flush()
			default:
			}
		case <-time.After(1 * time.Second):
			rl.phout.Flush()
		case <-ctx.Done():
			// Context is done, but we should read all data from source
			for {
				select {
				case r := <-rl.source:
					if err := rl.handle(r); err != nil {
						return err
					}
				default:
					break loop
				}
			}
		}
	}
	return nil
}

func NewPhoutResultListener(filename string) (rl ResultListener, err error) {
	var phoutFile *os.File
	if filename == "" {
		phoutFile = os.Stdout
	} else {
		phoutFile, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_SYNC, 0666)
	}
	writer := bufio.NewWriterSize(phoutFile, 1024*512) // 512 KB
	ch := make(chan *Sample, 65536)
	return &PhoutResultListener{
		source: ch,
		resultListener: resultListener{
			sink: ch,
		},
		phout:  writer,
		buffer: make([]byte, 0, 1024),
	}, nil
}

type phoutResultListeners map[string]ResultListener

func (prls phoutResultListeners) get(c *config.ResultListener) (ResultListener, error) {
	rl, ok := prls[c.Destination]
	if !ok {
		rl, err := NewPhoutResultListener(c.Destination)
		if err != nil {
			return nil, err
		}
		prls[c.Destination] = rl
		return rl, nil
	}
	return rl, nil
}

var defaultPhoutResultListeners = phoutResultListeners{}

// GetPhoutResultListener is not thread safe.
func GetPhoutResultListener(c *config.ResultListener) (ResultListener, error) {
	return defaultPhoutResultListeners.get(c)
}
