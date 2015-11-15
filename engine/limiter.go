package engine

import (
	"errors"
	"fmt"
	"time"
)

type Limiter interface {
	Start()
	Control() chan bool
}

type limiter struct {
	control chan bool
}

type periodicLimiter struct {
	limiter
	ticker *time.Ticker
}

func (l *limiter) Control() chan bool {
	return l.control
}

func (pl *periodicLimiter) Start() {
	go func() {
		pl.control <- true // first tick just after the start
		for range pl.ticker.C {
			pl.control <- true
		}
	}()
}

func NewPeriodicLimiter(period time.Duration) (l Limiter) {
	return &periodicLimiter{
		// timer-based limiters should have big enough cache
		limiter: limiter{make(chan bool, 65536)},
		ticker:  time.NewTicker(period),
	}
}

type batchLimiter struct {
	limiter
	master    Limiter
	batchSize int
}

func (bl *batchLimiter) Start() {
	bl.master.Start()
	go func() {
		for range bl.master.Control() {
			for i := 0; i < bl.batchSize; i++ {
				bl.control <- true
			}
		}
		close(bl.control)
	}()
}

func NewBatchLimiter(batchSize int, master Limiter) (l Limiter) {
	return &batchLimiter{
		limiter:   limiter{make(chan bool)},
		master:    master,
		batchSize: batchSize,
	}
}

type sizeLimiter struct {
	limiter
	master Limiter
	size   int
}

func (sl *sizeLimiter) Start() {
	sl.master.Start()
	go func() {
		c := sl.master.Control()
		for i := 0; i < sl.size; i++ {
			v, more := <-c
			if more {
				sl.control <- v
			} else {
				break
			}
		}
		close(sl.control)
	}()
}

func NewSizeLimiter(size int, master Limiter) (l Limiter) {
	return &sizeLimiter{
		limiter: limiter{make(chan bool)},
		master:  master,
		size:    size,
	}
}

type compositeLimiter struct {
	limiter
	steps []Limiter
}

func (cl *compositeLimiter) Start() {
	go func() {
		for _, l := range cl.steps {
			for range l.Control() {
				cl.control <- true
			}
		}
		close(cl.control)
	}()
}

func NewPeriodicLimiterFromConfig(c *LimiterConfig) (l Limiter, err error) {
	params := c.Parameters
	if params == nil {
		return nil, errors.New("Parameters not specified")
	}
	period, ok := params["Period"]
	if !ok {
		return nil, errors.New("Period not specified")
	}
	switch t := period.(type) {
	case float64:
		l = NewPeriodicLimiter(time.Duration(period.(float64)*1e3) * time.Millisecond)
	default:
		return nil, errors.New(fmt.Sprintf("Period is of the wrong type."+
			" Expected 'float64' got '%T'", t))
	}
	maxCount, ok := params["MaxCount"]
	if ok {
		mc, ok := maxCount.(float64)
		if !ok {
			return nil, errors.New(fmt.Sprintf("MaxCount is of the wrong type."+
				" Expected 'float64' got '%T'", maxCount))
		}
		l = NewSizeLimiter(int(mc), l)
	}
	batchSize, ok := params["BatchSize"]
	if ok {
		bs, ok := batchSize.(float64)
		if !ok {
			return nil, errors.New(fmt.Sprintf("BatchSize is of the wrong type."+
				" Expected 'float64' got '%T'", batchSize))
		}
		l = NewBatchLimiter(int(bs), l)
	}
	return l, nil
}

func NewCompositeLimiterFromConfig(c *LimiterConfig) (l Limiter, err error) {
	return nil, errors.New("Not implemented")
}

func NewLimiterFromConfig(c *LimiterConfig) (l Limiter, err error) {
	if c == nil {
		return
	}
	switch c.LimiterType {
	case "periodic":
		return NewPeriodicLimiterFromConfig(c)
	default:
		err = errors.New(fmt.Sprintf("No such limiter type: %s", c.LimiterType))
	}
	return
}
