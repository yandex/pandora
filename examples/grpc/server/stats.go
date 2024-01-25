package server

import (
	"sync"
	"sync/atomic"
)

func newStats(capacity int) *Stats {
	stats := Stats{
		auth200:       make(map[int64]uint64, capacity),
		auth200Mutex:  sync.Mutex{},
		auth400:       atomic.Uint64{},
		auth500:       atomic.Uint64{},
		list200:       make(map[int64]uint64, capacity),
		list200Mutex:  sync.Mutex{},
		list400:       atomic.Uint64{},
		list500:       atomic.Uint64{},
		order200:      make(map[int64]uint64, capacity),
		order200Mutex: sync.Mutex{},
		order400:      atomic.Uint64{},
		order500:      atomic.Uint64{},
	}
	return &stats
}

type Stats struct {
	auth200       map[int64]uint64
	auth200Mutex  sync.Mutex
	auth400       atomic.Uint64
	auth500       atomic.Uint64
	list200       map[int64]uint64
	list200Mutex  sync.Mutex
	list400       atomic.Uint64
	list500       atomic.Uint64
	order200      map[int64]uint64
	order200Mutex sync.Mutex
	order400      atomic.Uint64
	order500      atomic.Uint64
}

func (s *Stats) IncAuth400() {
	s.auth400.Add(1)
}

func (s *Stats) IncAuth500() {
	s.auth500.Add(1)
}

func (s *Stats) IncAuth200(userID int64) {
	s.auth200Mutex.Lock()
	s.auth200[userID]++
	s.auth200Mutex.Unlock()
}

func (s *Stats) IncList400() {
	s.list400.Add(1)
}

func (s *Stats) IncList500() {
	s.list500.Add(1)
}

func (s *Stats) IncList200(userID int64) {
	s.list200Mutex.Lock()
	s.list200[userID]++
	s.list200Mutex.Unlock()
}

func (s *Stats) IncOrder400() {
	s.order400.Add(1)
}

func (s *Stats) IncOrder500() {
	s.order500.Add(1)
}

func (s *Stats) IncOrder200(userID int64) {
	s.order200Mutex.Lock()
	s.order200[userID]++
	s.order200Mutex.Unlock()
}

func (s *Stats) Reset() {
	s.auth200Mutex.Lock()
	s.auth200 = map[int64]uint64{}
	s.auth200Mutex.Unlock()
	s.auth400.Store(0)
	s.auth500.Store(0)

	s.list200Mutex.Lock()
	s.list200 = map[int64]uint64{}
	s.list200Mutex.Unlock()
	s.list400.Store(0)
	s.list500.Store(0)

	s.order200Mutex.Lock()
	s.order200 = map[int64]uint64{}
	s.order200Mutex.Unlock()
	s.order400.Store(0)
	s.order500.Store(0)
}
