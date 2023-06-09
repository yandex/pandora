package base

import "sync"

type Decoder[R any] struct {
	Sink chan<- *Ammo[R]
	Pool *sync.Pool
}

func NewDecoder[R any](sink chan<- *Ammo[R]) Decoder[R] {
	return Decoder[R]{
		Sink: sink,
		Pool: &sync.Pool{New: func() any {
			return new(Ammo[R])
		}},
	}
}
