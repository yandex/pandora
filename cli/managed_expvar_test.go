package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken control pipe") }

func TestManagedExpvarEndpoint(t *testing.T) {
	var control strings.Builder
	stop, err := startManagedExpvar(&control)
	if err != nil {
		t.Fatal(err)
	}
	defer stop()

	var announcement struct {
		Version int    `json:"version"`
		Expvar  string `json:"expvar"`
	}
	if err := json.Unmarshal([]byte(control.String()), &announcement); err != nil {
		t.Fatalf("invalid control message %q: %v", control.String(), err)
	}
	if announcement.Version != 1 {
		t.Fatalf("protocol version = %d, want 1", announcement.Version)
	}
	host, port, err := net.SplitHostPort(announcement.Expvar)
	if err != nil || host != "127.0.0.1" || port == "0" || port == "" {
		t.Fatalf("managed endpoint = %q, want a selected loopback port: %v", announcement.Expvar, err)
	}

	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + announcement.Expvar + "/debug/vars")
	if err != nil {
		t.Fatalf("expvar endpoint is not serving: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expvar status = %d, want 200", response.StatusCode)
	}
	if _, err := io.ReadAll(response.Body); err != nil {
		t.Fatal(err)
	}
}

func TestManagedExpvarParallelEndpoints(t *testing.T) {
	var first, second strings.Builder
	stopFirst, err := startManagedExpvar(&first)
	if err != nil {
		t.Fatal(err)
	}
	defer stopFirst()
	stopSecond, err := startManagedExpvar(&second)
	if err != nil {
		t.Fatal(err)
	}
	defer stopSecond()
	if first.String() == second.String() {
		t.Fatalf("parallel managed endpoints collided: %s", first.String())
	}
}

func TestManagedExpvarControlWriteFailure(t *testing.T) {
	if stop, err := startManagedExpvar(failingWriter{}); err == nil || stop != nil {
		t.Fatalf("startManagedExpvar(broken pipe) returned stop=%t, err=%v, want false, error", stop != nil, err)
	}
}

func TestManagedExpvarStopsServing(t *testing.T) {
	var control strings.Builder
	stop, err := startManagedExpvar(&control)
	if err != nil {
		t.Fatal(err)
	}
	var announcement struct {
		Expvar string `json:"expvar"`
	}
	if err := json.Unmarshal([]byte(control.String()), &announcement); err != nil {
		stop()
		t.Fatal(err)
	}
	stop()
	client := &http.Client{Timeout: time.Second}
	if response, err := client.Get("http://" + announcement.Expvar + "/debug/vars"); err == nil {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
		t.Fatal("managed expvar is still serving after stop")
	}
}

func TestStandaloneExpvarDefaultDisabled(t *testing.T) {
	if DefaultConfig().Monitoring.Expvar.Enabled {
		t.Fatal("standalone expvar must remain disabled by default")
	}
}

func TestOpenManagedControlRejectsInvalidDescriptor(t *testing.T) {
	if _, err := openManagedControl(-1); err == nil {
		t.Fatal("negative control descriptor was accepted")
	}
	file, err := os.CreateTemp(t.TempDir(), "control")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	fd, err := syscall.Dup(int(file.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := openManagedControl(fd); err == nil {
		t.Fatal("regular file was accepted as control pipe")
	}
}

func TestOpenManagedControlAcceptsInheritedPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	defer writer.Close()
	fd, err := syscall.Dup(int(writer.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	control, err := openManagedControl(fd)
	if err != nil {
		t.Fatal(err)
	}
	defer control.Close()
	if int(control.Fd()) != fd {
		t.Fatalf("control fd = %d, want %d", control.Fd(), fd)
	}
}

func TestRunRejectsMissingControlFD(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestManagedExpvarHelperProcess", "--", "-managed-expvar-fd=3", "-")
	cmd.Env = append(os.Environ(), "PANDORA_MANAGED_EXPVAR_TEST_HELPER=missing-fd")
	output, err := cmd.CombinedOutput()
	if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 2 {
		t.Fatalf("missing control fd exit = %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "stat managed expvar file descriptor") {
		t.Fatalf("missing control fd error = %q", output)
	}
}

func TestRunReportsCapabilitiesWithoutConfig(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestManagedExpvarHelperProcess", "--", "-capabilities")
	cmd.Env = append(os.Environ(), "PANDORA_MANAGED_EXPVAR_TEST_HELPER=capabilities")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("capability probe failed: %v, output: %s", err, output)
	}
	if string(output) != "{\"managed_expvar_fd\":1}\n" {
		t.Fatalf("capability probe output = %q", output)
	}
}

func TestManagedExpvarInheritedPipe(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=TestManagedExpvarHelperProcess", "--")
	cmd.Env = append(os.Environ(), "PANDORA_MANAGED_EXPVAR_TEST_HELPER=inherited-fd")
	cmd.ExtraFiles = []*os.File{writer}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = writer.Close()
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		_ = writer.Close()
		t.Fatal(err)
	}
	_ = writer.Close()

	var announcement struct {
		Version int    `json:"version"`
		Expvar  string `json:"expvar"`
	}
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&announcement); err != nil {
		_ = stdin.Close()
		_ = cmd.Wait()
		t.Fatal(err)
	}
	if announcement.Version != 1 || !strings.HasPrefix(announcement.Expvar, "127.0.0.1:") {
		t.Errorf("invalid announcement: %+v", announcement)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		t.Errorf("control pipe did not close after one message: %v", err)
	}
	client := &http.Client{Timeout: time.Second}
	response, err := client.Get("http://" + announcement.Expvar + "/debug/vars")
	if err != nil {
		t.Error(err)
	} else {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}
	if err := stdin.Close(); err != nil {
		t.Error(err)
	}
	if err := cmd.Wait(); err != nil {
		t.Error(err)
	}
}

func TestManagedExpvarHelperProcess(t *testing.T) {
	mode := os.Getenv("PANDORA_MANAGED_EXPVAR_TEST_HELPER")
	if mode == "" {
		return
	}
	flag.CommandLine = flag.NewFlagSet("pandora", flag.ExitOnError)
	switch mode {
	case "missing-fd":
		os.Args = []string{"pandora", "-managed-expvar-fd=3", "-"}
	case "capabilities":
		os.Args = []string{"pandora", "-capabilities"}
	case "inherited-fd":
		control, err := openManagedControl(3)
		if err != nil {
			os.Exit(4)
		}
		stop, err := startManagedExpvar(control)
		if err != nil {
			os.Exit(5)
		}
		if err := control.Close(); err != nil {
			os.Exit(6)
		}
		_, _ = io.Copy(io.Discard, os.Stdin)
		stop()
		os.Exit(0)
	default:
		os.Exit(3)
	}
	Run()
	os.Exit(0)
}
