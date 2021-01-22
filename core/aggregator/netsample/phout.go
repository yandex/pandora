package netsample

import (
	"bufio"
	"context"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coreutil"
)

type PhoutConfig struct {
	Destination     string                    // Destination file name
	ID              bool                      // Print ammo ids if true.
	FlushTime       time.Duration             `config:"flush-time"`
	SampleQueueSize int                       `config:"sample-queue-size"`
	Buffer          coreutil.BufferSizeConfig `config:",squash"`
}

func DefaultPhoutConfig() PhoutConfig {
	return PhoutConfig{
		FlushTime:       time.Second,
		SampleQueueSize: 256 * 1024,
		Buffer: coreutil.BufferSizeConfig{
			BufferSize: 8 * datasize.MB,
		},
	}
}

func NewPhout(fs afero.Fs, conf PhoutConfig) (a Aggregator, err error) {
	filename := conf.Destination
	var file afero.File = os.Stdout
	if filename != "" {
		file, err = fs.Create(conf.Destination)
	}
	if err != nil {
		err = errors.Wrap(err, "phout output file open failed")
		return
	}
	a = &phoutAggregator{
		config: conf,
		sink:   make(chan *Sample, conf.SampleQueueSize),
		writer: bufio.NewWriterSize(file, conf.Buffer.BufferSizeOrDefault()),
		buf:    make([]byte, 0, 1024),
		file:   file,
	}
	return
}

type phoutAggregator struct {
	config PhoutConfig
	sink   chan *Sample
	writer *bufio.Writer
	buf    []byte
	file   io.Closer
}

func (a *phoutAggregator) Report(s *Sample) { a.sink <- s }

func (a *phoutAggregator) Run(ctx context.Context, _ core.AggregatorDeps) error {
	shouldFlush := time.NewTicker(1 * time.Second)
	defer func() {
		_ = a.writer.Flush()
		_ = a.file.Close()
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
				_ = a.writer.Flush()
			default:
			}
		case <-time.After(1 * time.Second):
			_ = a.writer.Flush()
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
	a.buf = appendPhout(s, a.buf, a.config.ID)
	a.buf = append(a.buf, '\n')
	_, err := a.writer.Write(a.buf)
	a.buf = a.buf[:0]
	releaseSample(s)
	return err
}

const phoutDelimiter = '\t'

func appendPhout(s *Sample, dst []byte, id bool) []byte {
	dst = appendTimestamp(s.timeStamp, dst)
	dst = append(dst, phoutDelimiter)
	dst = append(dst, s.tags...)
	if id {
		dst = append(dst, '#')
		dst = strconv.AppendInt(dst, int64(s.ID()), 10)
	}
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
