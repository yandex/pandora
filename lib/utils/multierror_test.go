package utils

import (
	"errors"
	"reflect"
	"testing"
)

func TestError_Impl(t *testing.T) {
	var _ error = new(MultiError)
}

func TestErrorError_custom(t *testing.T) {
	errors := []error{
		errors.New("foo"),
		errors.New("bar"),
	}

	fn := func(es []error) string {
		return "foo"
	}

	multi := &MultiError{Errors: errors, ErrorFormat: fn}
	if multi.Error() != "foo" {
		t.Fatalf("bad: %s", multi.Error())
	}
}

func TestErrorError_default(t *testing.T) {
	expected := "2 error(s) occurred:\n\n* foo\n* bar"

	errors := []error{
		errors.New("foo"),
		errors.New("bar"),
	}

	multi := &MultiError{Errors: errors}
	if multi.Error() != expected {
		t.Fatalf("bad: %s", multi.Error())
	}
}

func TestErrorErrorOrNil(t *testing.T) {
	err := new(MultiError)
	if err.ErrorOrNil() != nil {
		t.Fatalf("bad: %#v", err.ErrorOrNil())
	}

	err.Errors = []error{errors.New("foo")}
	if v := err.ErrorOrNil(); v == nil {
		t.Fatal("should not be nil")
	} else if !reflect.DeepEqual(v, err) {
		t.Fatalf("bad: %#v", v)
	}
}

func TestErrorWrappedErrors(t *testing.T) {
	errors := []error{
		errors.New("foo"),
		errors.New("bar"),
	}

	multi := &MultiError{Errors: errors}
	if !reflect.DeepEqual(multi.Errors, multi.WrappedErrors()) {
		t.Fatalf("bad: %s", multi.WrappedErrors())
	}
}

func TestListFormatFunc(t *testing.T) {
	expected := "2 error(s) occurred:\n\n* foo\n* bar"

	errors := []error{
		errors.New("foo"),
		errors.New("bar"),
	}

	actual := ListFormatFunc(errors)
	if actual != expected {
		t.Fatalf("bad: %#v", actual)
	}
}
