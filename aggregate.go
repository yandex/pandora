package main

import (
	"log"
)

type Sample interface {
	PhoutSample() *PhoutSample
	String() string
}

type ResultListener interface {
	Start()
	Sink() chan Sample
}

type resultListener struct {
	sink chan Sample
}

func (rl *resultListener) Sink() chan Sample {
	return rl.sink
}

type LoggingResultListener struct {
	resultListener
}

func (rl *LoggingResultListener) Start() {
	go func() {
		for r := range rl.sink {
			log.Println(r)
		}
	}()
}

func NewLoggingResultListener() ResultListener {
	return &LoggingResultListener{
		resultListener: resultListener{
			sink: make(chan Sample, 32),
		},
	}
}
