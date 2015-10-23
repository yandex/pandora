package main

import (
	"errors"
	"fmt"
	"log"
	"os"
)

type Sample interface {
	String() string
}

type PhoutSample struct {
	ts             float64
	tag            string
	rt             int
	connect        int
	send           int
	latency        int
	receive        int
	interval_event int
	egress         int
	igress         int
	netCode        int
	protoCode      int
}

func (ps *PhoutSample) String() string {
	return fmt.Sprintf(
		"%.3f\t%s\t%d\t"+
			"%d\t%d\t"+
			"%d\t%d\t"+
			"%d\t"+
			"%d\t%d\t"+
			"%d\t%d",
		ps.ts, ps.tag, ps.rt,
		ps.connect, ps.send,
		ps.latency, ps.receive,
		ps.interval_event,
		ps.egress, ps.igress,
		ps.netCode, ps.protoCode,
	)
}

type PhantomCompatible interface {
	Sample
	PhoutSample() *PhoutSample
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

type PhoutResultListener struct {
	resultListener
	phout *os.File
}

func (rl *PhoutResultListener) Start() {
	go func() {
		for r := range rl.sink {
			pc, ok := r.(PhantomCompatible)
			if ok {
				rl.phout.WriteString(fmt.Sprintf("%s\n", pc.PhoutSample()))
			} else {
				log.Panic("Not phantom compatible sample")
				return
			}
		}
	}()
}

func NewPhoutResultListener(filename string) (rl ResultListener, err error) {
	var phoutFile *os.File
	if filename == "" {
		phoutFile = os.Stdout
	} else {
		phoutFile, err = os.Create(filename)
	}
	return &PhoutResultListener{
		resultListener: resultListener{
			sink: make(chan Sample, 32),
		},
		phout: phoutFile,
	}, nil
}

type phoutResultListenerFactory struct {
	listeners map[string]ResultListener
}

func (prls *phoutResultListenerFactory) Create(c *ResultListenerConfig) (rl ResultListener, err error) {
	rl, ok := prls.listeners[c.Destination]
	if !ok {
		rl, err = NewPhoutResultListener(c.Destination)
		if err != nil {
			return nil, err
		} else {
			prls.listeners[c.Destination] = rl
		}
	}
	return
}

var PhoutResultListenerFactory *phoutResultListenerFactory = &phoutResultListenerFactory{
	make(map[string]ResultListener),
}

func NewResultListenerFromConfig(c *ResultListenerConfig) (rl ResultListener, err error) {
	if c == nil {
		return
	}
	switch c.ListenerType {
	case "log/simple":
		rl, err = NewLoggingResultListener()
	case "log/phout":
		rl, err = PhoutResultListenerFactory.Create(c)
	default:
		err = errors.New(fmt.Sprintf("No such listener type: %s", c.ListenerType))
	}
	return
}
