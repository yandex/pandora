package datasource

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/core/coretest"
)

func TestFileSource(t *testing.T) {
	const filename = "/xxx/yyy"
	fs := afero.NewMemMapFs()
	source := NewFile(fs, FileConfig{Path: filename})
	coretest.AssertSourceEqualFile(t, fs, filename, source)
}

func TestStdin(t *testing.T) {
	coretest.AssertSourceEqualStdStream(t, &os.Stdout, NewStdin)
}
