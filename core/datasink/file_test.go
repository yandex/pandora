package datasink

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/core/coretest"
)

func TestFileSink(t *testing.T) {
	const filename = "/xxx/yyy"
	fs := afero.NewMemMapFs()
	sink := NewFile(fs, FileConfig{Path: filename})
	coretest.AssertSinkEqualFile(t, fs, filename, sink)
}

func TestStdout(t *testing.T) {
	coretest.AssertSinkEqualStdStream(t, &os.Stdout, NewStdout)
}

func TestStderr(t *testing.T) {
	coretest.AssertSinkEqualStdStream(t, &os.Stderr, NewStderr)
}
