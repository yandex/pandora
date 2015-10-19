package main

import (
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
		for range pl.ticker.C {
			pl.control <- true
		}
	}()
}

func NewPeriodicLimiter(period time.Duration) (l Limiter) {
	return &periodicLimiter{
		limiter: limiter{make(chan bool, 1024)},
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
	}()
}

func NewBatchLimiter(batchSize int, master Limiter) (l Limiter) {
	return &batchLimiter{
		limiter:   limiter{make(chan bool, 1024)},
		master:    master,
		batchSize: batchSize,
	}
}
