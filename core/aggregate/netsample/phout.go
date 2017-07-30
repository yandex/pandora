package netsample

import (
	"bufio"
	"context"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/spf13/afero"
)

func GetPhout(fs afero.Fs, conf PhoutConfig) (Aggregator, error) {
	return defaultPhoutAggregator.get(fs, conf)
}

type PhoutConfig struct {
	Destination string
}

type phoutAggregator struct {
	sink   chan *Sample
	writer *bufio.Writer
	buf    []byte
	file   io.Closer
}

var _ Aggregator = (*phoutAggregator)(nil)

func (a *phoutAggregator) Report(s *Sample) { a.sink <- s }

func (a *phoutAggregator) Run(ctx context.Context) error {
	shouldFlush := time.NewTicker(1 * time.Second)
	defer func() {
		a.writer.Flush()
		a.file.Close()
		shouldFlush.Stop()
	}()
loop:
	for {
		select {
		case r := <-a.sink:
			if err := a.handle(r); err != nil {
				return err
			}
			select {
			case <-shouldFlush.C:
				a.writer.Flush()
			default:
			}
		case <-time.After(1 * time.Second):
			a.writer.Flush()
		case <-ctx.Done():
			// Context is done, but we should read all data from sink
			for {
				select {
				case r := <-a.sink:
					if err := a.handle(r); err != nil {
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

func (a *phoutAggregator) handle(s *Sample) error {
	a.buf = appendPhout(s, a.buf)
	a.buf = append(a.buf, '\n')
	_, err := a.writer.Write(a.buf)
	a.buf = a.buf[:0]
	releaseSample(s)
	return err
}

var defaultPhoutAggregator = newPhoutResultListeners()

type phoutAggregators struct {
	sync.Mutex
	aggregators map[string]Aggregator
}

func newPhout(fs afero.Fs, conf PhoutConfig) (a *phoutAggregator, err error) {
	filename := conf.Destination
	var file afero.File = os.Stdout
	if filename != "" {
		file, err = fs.OpenFile(filename, os.O_APPEND|os.O_WRONLY|os.O_CREATE|os.O_SYNC, 0666)
	}
	if err != nil {
		return
	}
	a = &phoutAggregator{
		sink:   make(chan *Sample, 32*1024),
		writer: bufio.NewWriterSize(file, 512*1024),
		buf:    make([]byte, 0, 1024),
		file:   file,
	}
	return
}

func newPhoutResultListeners() *phoutAggregators {
	return &phoutAggregators{aggregators: make(map[string]Aggregator)}
}

func (l *phoutAggregators) get(fs afero.Fs, conf PhoutConfig) (Aggregator, error) {
	dest := conf.Destination
	l.Lock()
	defer l.Unlock()
	rl, ok := l.aggregators[dest]
	if !ok {
		rl, err := newPhout(fs, conf)
		if err != nil {
			return nil, err
		}
		l.aggregators[dest] = rl
		return rl, nil
	}
	return rl, nil
}

const phoutDelimiter = '\t'

func appendPhout(s *Sample, dst []byte) []byte {
	dst = appendTimestamp(s.timeStamp, dst)
	dst = append(dst, phoutDelimiter)
	dst = append(dst, s.tags...)
	for _, v := range s.fields {
		dst = append(dst, phoutDelimiter)
		dst = strconv.AppendInt(dst, int64(v), 10)
	}
	return dst
}

func appendTimestamp(ts time.Time, dst []byte) []byte {
	// Append time stamp in phout format. Example: 1335524833.562
	// Algorithm: append milliseconds string, than insert dot in right place.
	dst = strconv.AppendInt(dst, ts.UnixNano()/1e6, 10)
	dotIndex := len(dst) - 3
	dst = append(dst, 0)
	for i := len(dst) - 1; i > dotIndex; i-- {
		dst[i] = dst[i-1]
	}
	dst[dotIndex] = '.'
	return dst
}
