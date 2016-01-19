package ammo

import (
	"expvar"
	"log"
	"strconv"
	"sync/atomic"
	"time"
)

type Counter struct {
	i int64
}

func (c *Counter) String() string {
	return strconv.FormatInt(atomic.LoadInt64(&c.i), 10)
}

func (c *Counter) Add(delta int64) {
	atomic.AddInt64(&c.i, delta)
}

func (c *Counter) Set(value int64) {
	atomic.StoreInt64(&c.i, value)
}

func (c *Counter) Get() int64 {
	return atomic.LoadInt64(&c.i)
}

func NewCounter(name string) *Counter {
	v := &Counter{}
	expvar.Publish(name, v)
	return v
}

var (
	evPassesLeft = NewCounter("ammo_PassesLeft")
)

func init() {
	go func() {
		passesLeft := evPassesLeft.Get()
		for _ = range time.NewTicker(1 * time.Second).C {
			if passesLeft < 0 {
				return // infinite number of passes
			}
			newPassesLeft := evPassesLeft.Get()
			if newPassesLeft != passesLeft {
				log.Printf("[AMMO] passes left: %d", newPassesLeft)
				passesLeft = newPassesLeft
			}
		}
	}()
}
