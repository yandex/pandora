package datasink

import (
	"io"
	"os"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/core"
)

// TODO(skipor): gzip on flag

type FileConfig struct {
	Path string `config:"path" validate:"required"`
}

func NewFile(fs afero.Fs, conf FileConfig) core.DataSink {
	return &fileSink{afero.Afero{Fs: fs}, conf}
}

type fileSink struct {
	fs   afero.Afero
	conf FileConfig
}

func (s *fileSink) OpenSink() (wc io.WriteCloser, err error) {
	return s.fs.OpenFile(s.conf.Path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
}

func NewStdout() core.DataSink {
	return hideCloseFileSink{os.Stdout}
}

func NewStderr() core.DataSink {
	return hideCloseFileSink{os.Stderr}
}

type hideCloseFileSink struct{ afero.File }

func (f hideCloseFileSink) OpenSink() (wc io.WriteCloser, err error) {
	return f, nil
}

func (f hideCloseFileSink) Close() error { return nil }
