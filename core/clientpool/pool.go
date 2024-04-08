package clientpool

import (
	"errors"
	"sync/atomic"
)

func New[T any](size int) (*Pool[T], error) {
	if size <= 0 {
		return nil, errors.New("pool size must be greater than zero")
	}
	return &Pool[T]{
		pool: make([]T, 0, size),
	}, nil
}

type Pool[T any] struct {
	pool []T
	i    atomic.Uint64
}

func (p *Pool[T]) Add(conn T) {
	p.pool = append(p.pool, conn)
}

func (p *Pool[T]) Next() T {
	if len(p.pool) == 0 {
		var zero T
		return zero
	}
	i := p.i.Add(1)
	return p.pool[int(i)%len(p.pool)]
}
