package main

import (
	"fmt"
	"log"
)

type LogGun struct{}

type LogJob struct {
	message string
}

func (l *LogGun) Run(j Job, results chan<- Sample) {
	log.Println("Log message: ", j.(*LogJob).message)
	results <- &DummySample{0}
}

type DummySample struct {
	value int
}

func (ds *DummySample) PhoutSample() *PhoutSample {
	return &PhoutSample{}
}

func (ds *DummySample) String() string {
	return fmt.Sprintf("My value is %d", ds.value)
}
