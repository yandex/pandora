package datasource

import (
	"io"
	"os"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/core"
)

// TODO(skipor): auto unzip with option to turn this behaviour off.

type FileConfig struct {
	Path string `config:"path" validate:"required"`
}

func NewFile(fs afero.Fs, conf FileConfig) core.DataSource {
	return &fileSource{afero.Afero{Fs: fs}, conf}
}

type fileSource struct {
	fs   afero.Afero
	conf FileConfig
}

func (s *fileSource) OpenSource() (wc io.ReadCloser, err error) {
	return s.fs.Open(s.conf.Path)
}

func NewStdin() core.DataSource {
	return hideCloseFileSource{os.Stdin}
}

type hideCloseFileSource struct{ afero.File }

func (f hideCloseFileSource) OpenSource() (wc io.ReadCloser, err error) {
	return f, nil
}

func (f hideCloseFileSource) Close() error { return nil }
