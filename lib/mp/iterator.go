package mp

import (
	"math/rand"
	"sync"
	"sync/atomic"
)

type Iterator interface {
	Next(segment string) int
	Rand(length int) int
}

func NewNextIterator(seed int64) *NextIterator {
	return &NextIterator{
		gs:  make(map[string]*atomic.Uint64),
		rnd: rand.New(rand.NewSource(seed)),
	}
}

type NextIterator struct {
	mx  sync.Mutex
	gs  map[string]*atomic.Uint64
	rnd *rand.Rand
}

func (n *NextIterator) Rand(length int) int {
	return n.rnd.Intn(length)
}

func (n *NextIterator) Next(segment string) int {
	a, ok := n.gs[segment]
	if !ok {
		n.mx.Lock()
		n.gs[segment] = &atomic.Uint64{}
		n.mx.Unlock()
		return 0
	}
	add := a.Add(1)
	return int(add)
}
