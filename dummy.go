package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type LogGun struct{}

type LogAmmo struct {
	Message string
}

type LogAmmoJsonDecoder struct{}

func (la *LogAmmoJsonDecoder) FromString(jsonDoc string) (a Ammo, err error) {
	a = &LogAmmo{}
	err = json.Unmarshal([]byte(jsonDoc), a)
	return
}

func (l *LogGun) Run(a Ammo, results chan<- Sample) {
	log.Println("Log message: ", a.(*LogAmmo).Message)
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
