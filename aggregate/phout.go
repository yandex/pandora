package aggregate

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"sync"
	"time"
)

func GetPhoutResultListener(conf PhoutResultListenerConfig) (ResultListener, error) {
	return defaultPhoutResultListeners.get(conf)
}

type PhoutResultListenerConfig struct {
	Destination string
}

type phoutResultListener struct {
	resultListener
	source <-chan *Sample
	phout  *bufio.Writer
	buf    []byte
}

var _ ResultListener = (*phoutResultListener)(nil)

func (rl *phoutResultListener) Start(ctx context.Context) error {
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

func (rl *phoutResultListener) handle(s *Sample) error {
	rl.buf = appendPhout(s, rl.buf)
	rl.buf = append(rl.buf, '\n')
	_, err := rl.phout.Write(rl.buf)
	rl.buf = rl.buf[:0]
	ReleaseSample(s)
	return err
}

var defaultPhoutResultListeners = newPhoutResultListeners()

type phoutResultListeners struct {
	sync.Mutex
	listeners map[string]ResultListener
}

func newPhoutResultListener(conf PhoutResultListenerConfig) (rl ResultListener, err error) {
	filename := conf.Destination
	var phoutFile *os.File
	if filename == "" {
		phoutFile = os.Stdout
	} else {
		phoutFile, err = os.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_SYNC, 0666)
	}
	writer := bufio.NewWriterSize(phoutFile, 1024*512) // 512 KB
	ch := make(chan *Sample, 65536)
	return &phoutResultListener{
		source: ch,
		resultListener: resultListener{
			sink: ch,
		},
		phout: writer,
		buf:   make([]byte, 0, 1024),
	}, nil
}

func newPhoutResultListeners() *phoutResultListeners {
	return &phoutResultListeners{listeners: make(map[string]ResultListener)}
}

func (l *phoutResultListeners) get(conf PhoutResultListenerConfig) (ResultListener, error) {
	dest := conf.Destination
	l.Lock()
	defer l.Unlock()
	rl, ok := l.listeners[dest]
	if !ok {
		rl, err := newPhoutResultListener(conf)
		if err != nil {
			return nil, err
		}
		l.listeners[dest] = rl
		return rl, nil
	}
	return rl, nil
}

func appendPhout(s *Sample, dst []byte) []byte {
	const phoutDelimiter = '\t'
	// Append time stamp in phout format. Example: 1335524833.562
	// Algorithm: append milliseconds string, than insert dot in right place.
	dst = strconv.AppendInt(dst, s.timeStamp.UnixNano()/1e6, 10)
	dotIndex := len(dst) - 3
	dst = append(dst, 0) // Add byte for dot.
	// Shift right last three digits, to get space for dot.
	for i := len(dst) - 1; i > dotIndex; i-- {
		dst[i] = dst[i-1]
	}
	dst[dotIndex] = '.'
	dst = append(dst, phoutDelimiter)

	dst = append(dst, s.tags...)
	for _, v := range s.fields {
		dst = append(dst, phoutDelimiter)
		dst = strconv.AppendInt(dst, int64(v), 10)
	}
	return dst
}
