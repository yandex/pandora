package datasource

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/ioutil2"
)

func NewBuffer(buf *bytes.Buffer) core.DataSource {
	return buffer{Buffer: buf}
}

type buffer struct {
	*bytes.Buffer
	ioutil2.NopCloser
}

func (b buffer) OpenSource() (wc io.ReadCloser, err error) {
	return b, nil
}

// NewReader returns dummy core.DataSource that returns it on OpenSource call, wrapping it
// ioutil.NopCloser if r is not io.Closer.
// NOTE(skipor): such wrapping hides Seek and other methods that can be used.
func NewReader(r io.Reader) core.DataSource {
	return &readerSource{r}
}

type readerSource struct {
	source io.Reader
}

func (r *readerSource) OpenSource() (rc io.ReadCloser, err error) {
	if rc, ok := r.source.(io.ReadCloser); ok {
		return rc, nil
	}
	// Need to add io.Closer, but don't want to hide seeker.
	rs, ok := r.source.(io.ReadSeeker)
	if ok {
		return &struct {
			io.ReadSeeker
			ioutil2.NopCloser
		}{ReadSeeker: rs}, nil
	}
	return ioutil.NopCloser(r.source), nil
}

func NewString(s string) core.DataSource {
	return &stringSource{Reader: strings.NewReader(s)}
}

type stringSource struct {
	*strings.Reader
	ioutil2.NopCloser
}

func (s stringSource) OpenSource() (rc io.ReadCloser, err error) {
	return s, nil
}

type InlineConfig struct {
	Data string `validate:"required"`
}

func NewInline(conf InlineConfig) core.DataSource {
	return NewString(conf.Data)
}

// TODO(skipor): InMemory DataSource, that reads all nested source data in open to buffer.
