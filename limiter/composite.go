package limiter

import (
	"fmt"

	"github.com/yandex/pandora/config"
	"golang.org/x/net/context"
)

type composite struct {
	limiter
	steps []Limiter
}

var _ Limiter = (*composite)(nil) // check interface

func (cl *composite) Start(ctx context.Context) error {
	defer close(cl.control)
outer_loop:
	for _, l := range cl.steps {
	inner_loop:
		for {
			select {
			case _, more := <-l.Control():
				if !more {
					break inner_loop
				}
				select {
				case cl.control <- struct{}{}:
				case <-ctx.Done():
					break outer_loop
				}
			case <-ctx.Done():
				break outer_loop
			}
		}

	}

	return nil
}

func NewCompositeFromConfig(c *config.Limiter) (l Limiter, err error) {
	return nil, fmt.Errorf("Not implemented")
}
