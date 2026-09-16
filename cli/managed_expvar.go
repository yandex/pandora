package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"go.uber.org/zap"
)

func openManagedControl(fd int) (*os.File, error) {
	if fd < 3 {
		return nil, fmt.Errorf("managed expvar file descriptor must be at least 3: %d", fd)
	}
	control := os.NewFile(uintptr(fd), "managed-expvar-control")
	if control == nil {
		return nil, fmt.Errorf("invalid managed expvar file descriptor: %d", fd)
	}
	info, err := control.Stat()
	if err != nil {
		_ = control.Close()
		return nil, fmt.Errorf("stat managed expvar file descriptor %d: %w", fd, err)
	}
	if info.Mode()&os.ModeNamedPipe == 0 {
		_ = control.Close()
		return nil, fmt.Errorf("managed expvar file descriptor %d is not a pipe", fd)
	}
	return control, nil
}

// startManagedExpvar reports the selected loopback endpoint to the parent before returning.
func startManagedExpvar(control io.Writer) (func(), error) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for managed expvar: %w", err)
	}

	announcement := struct {
		Version int    `json:"version"`
		Expvar  string `json:"expvar"`
	}{Version: 1, Expvar: listener.Addr().String()}
	if err := json.NewEncoder(control).Encode(announcement); err != nil {
		if closeErr := listener.Close(); closeErr != nil {
			zap.L().Warn("Cannot close managed expvar listener", zap.Error(closeErr))
		}
		return nil, fmt.Errorf("report managed expvar endpoint: %w", err)
	}

	server := &http.Server{Handler: http.DefaultServeMux}
	go func() {
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			zap.L().Fatal("Managed expvar server failed", zap.Error(err))
		}
	}()
	return func() {
		if err := server.Close(); err != nil {
			zap.L().Warn("Cannot stop managed expvar server", zap.Error(err))
		}
	}, nil
}
