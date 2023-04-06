package errutil

import (
	"context"
	"errors"
	"fmt"
	"testing"

	pkgerrors "github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestJoin(t *testing.T) {
	type args struct {
	}
	err1 := errors.New("error message")
	err2 := errors.New("error message 2")
	tests := []struct {
		name        string
		err1        error
		err2        error
		wantMessage string
		wantErr     error
		wantNil     bool
	}{
		{
			name:        "nil result",
			err1:        nil,
			err2:        nil,
			wantMessage: "",
			wantNil:     true,
		},
		{
			name:        "first error only",
			err1:        err1,
			err2:        nil,
			wantMessage: "error message",
			wantNil:     false,
		},
		{
			name:        "second error only",
			err1:        nil,
			err2:        err2,
			wantMessage: "error message 2",
			wantNil:     false,
		},
		{
			name:        "two errors",
			err1:        err1,
			err2:        err2,
			wantMessage: "2 errors occurred:\n\t* error message\n\t* error message 2\n\n",
			wantNil:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Join(tt.err1, tt.err2)
			if tt.wantNil {
				require.NoError(t, err)
				return
			}
			require.Equal(t, tt.wantMessage, err.Error())
		})
	}
}

func TestIsCtxError(t *testing.T) {
	canceledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	tests := []struct {
		name               string
		err                error
		wantCanceledCtx    bool
		wantNotCanceledCtx bool
	}{
		{
			name:               "nil error",
			err:                nil,
			wantCanceledCtx:    true,
			wantNotCanceledCtx: true,
		},
		{
			name:               "context error",
			err:                context.Canceled,
			wantCanceledCtx:    true,
			wantNotCanceledCtx: false,
		},
		{
			name:               "caused by context error",
			err:                pkgerrors.Wrap(context.Canceled, "new err"),
			wantCanceledCtx:    true,
			wantNotCanceledCtx: false,
		},
		{
			name:               "default error wrapping has defferent result",
			err:                fmt.Errorf("new err 2 %w", context.Canceled),
			wantCanceledCtx:    false,
			wantNotCanceledCtx: false,
		},
		{
			name:               "usual error",
			err:                errors.New("new err"),
			wantCanceledCtx:    false,
			wantNotCanceledCtx: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			canceledResult := IsCtxError(canceledCtx, tt.err)
			require.Equal(t, tt.wantCanceledCtx, canceledResult)

			notCanceledResult := IsCtxError(context.Background(), tt.err)
			require.Equal(t, tt.wantNotCanceledCtx, notCanceledResult)
		})
	}
}
