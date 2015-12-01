package utils

import "golang.org/x/net/context"

// Promise is a basic promise implementation: it wraps calls a function in a goroutine,
// and returns a channel which will later return the function's return value.
func Promise(f PromiseFunc) chan error {
	ch := make(chan error, 1)
	go func() {
		ch <- f()
	}()
	return ch
}

// Promise is a basic promise implementation: it wraps calls a function in a goroutine,
// and returns a channel which will later return the function's return value.
func PromiseCtx(ctx context.Context, f func(ctx context.Context) error) chan error {
	return Promise(func() error { return f(ctx) })
}

type Promises []chan error

type PromiseFunc func() error

// group asyncs and return grouped err after all async is finished
func (promises Promises) All() chan error {
	// TODO: wait in select for all promises at once. Because we can possible have a deadlock right now.
	return Promise(func() error {
		var result *MultiError
		for _, a := range promises {
			err := <-a
			if err != nil {
				result = AppendMulti(result, err)
			}
		}
		return result.ErrorOrNil()
	})

}
