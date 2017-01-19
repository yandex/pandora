package spdy

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestExamples(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	examples, err := ioutil.ReadDir("examples")
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)

	for _, example := range examples {
		err = os.Chdir(filepath.Join(cwd, "examples", example.Name()))
		if err != nil {
			t.Error(err)
			continue
		}

		buf.Reset()

		cmd := exec.Command("go", "build", "-o", "test")
		cmd.Stderr = buf
		if err = cmd.Run(); err != nil {
			t.Errorf("Example %q failed to compile:\n%s", example.Name(), buf.String())
			continue
		}

		if err = os.Remove("test"); err != nil {
			t.Error(err)
		}
	}
}
