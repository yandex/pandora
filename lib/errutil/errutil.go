package errutil

import (
	"context"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

type StackTracer interface {
	StackTrace() errors.StackTrace
}

// FIXME(skipor): test
func Join(err1, err2 error) error {
	switch {
	case err1 == nil:
		return err2
	case err2 == nil:
		return err1
	default:
		return multierror.Append(err1, err2)
	}
}

func IsCtxError(ctx context.Context, err error) bool {
	if err == nil {
		return true
	}
	return ctx.Err() == errors.Cause(err)
}
