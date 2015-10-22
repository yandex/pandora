package main

import (
	"errors"
	"fmt"
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

func NewLoggingResultListener() (rl ResultListener, err error) {
	return &LoggingResultListener{
		resultListener: resultListener{
			sink: make(chan Sample, 32),
		},
	}, nil
}

func NewResultListenerFromConfig(c *ResultListenerConfig) (rl ResultListener, err error) {
	if c == nil {
		return
	}
	switch c.ListenerType {
	case "log/simple":
		rl, err = NewLoggingResultListener()
	case "log/phout":
		err = errors.New(fmt.Sprintf("phout not implemented"))
	default:
		err = errors.New(fmt.Sprintf("No such listener type: %s", c.ListenerType))
	}
	return
}
