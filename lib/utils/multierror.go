package utils

// Multierror was takenfrom https://github.com/hashicorp/go-multierror

import (
	"fmt"
	"strings"
)

// MultiError is an error type to track multiple errors. This is used to
// accumulate errors in cases and return them as a single "error".
type MultiError struct {
	Errors      []error
	ErrorFormat ErrorFormatFunc
}

func (e *MultiError) Error() string {
	fn := e.ErrorFormat
	if fn == nil {
		fn = ListFormatFunc
	}

	return fn(e.Errors)
}

// ErrorOrNil returns an error interface if this Error represents
// a list of errors, or returns nil if the list of errors is empty. This
// function is useful at the end of accumulation to make sure that the value
// returned represents the existence of errors.
func (e *MultiError) ErrorOrNil() error {
	if e == nil {
		return nil
	}
	if len(e.Errors) == 0 {
		return nil
	}

	return e
}

func (e *MultiError) GoString() string {
	return fmt.Sprintf("*%#v", *e)
}

// WrappedErrors returns the list of errors that this Error is wrapping.
// It is an implementatin of the errwrap.Wrapper interface so that
// multierror.Error can be used with that library.
//
// This method is not safe to be called concurrently and is no different
// than accessing the Errors field directly. It is implementd only to
// satisfy the errwrap.Wrapper interface.
func (e *MultiError) WrappedErrors() []error {
	return e.Errors
}

// ErrorFormatFunc is a function callback that is called by Error to
// turn the list of errors into a string.
type ErrorFormatFunc func([]error) string

// ListFormatFunc is a basic formatter that outputs the number of errors
// that occurred along with a bullet point list of the errors.
func ListFormatFunc(es []error) string {
	points := make([]string, len(es))
	for i, err := range es {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d error(s) occurred:\n\n%s",
		len(es), strings.Join(points, "\n"))
}

// AppendMulti is a helper function that will append more errors
// onto an Error in order to create a larger multi-error.
//
// If err is not a multierror.Error, then it will be turned into
// one. If any of the errs are multierr.Error, they will be flattened
// one level into err.
func AppendMulti(err error, errs ...error) *MultiError {
	switch err := err.(type) {
	case *MultiError:
		// Typed nils can reach here, so initialize if we are nil
		if err == nil {
			err = new(MultiError)
		}

		// Go through each error and flatten
		for _, e := range errs {
			switch e := e.(type) {
			case *MultiError:
				err.Errors = append(err.Errors, e.Errors...)
			default:
				err.Errors = append(err.Errors, e)
			}
		}

		return err
	default:
		newErrs := make([]error, 0, len(errs)+1)
		if err != nil {
			newErrs = append(newErrs, err)
		}
		newErrs = append(newErrs, errs...)

		return AppendMulti(&MultiError{}, newErrs...)
	}
}
