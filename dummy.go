package main

import (
	"encoding/json"
	"fmt"
	"log"
)

type LogGun struct{}

type LogAmmo struct {
	message string
}

type LogAmmoJsonDecoder struct{}

func (la *LogAmmoJsonDecoder) FromString(jsonDoc string) (a Ammo, err error) {
	a = &LogAmmo{}
	err = json.Unmarshal([]byte(jsonDoc), a)
	return
}

func (l *LogGun) Run(a Ammo, results chan<- Sample) {
	log.Println("Log message: ", a.(*LogAmmo).message)
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

type LogAmmoProvider struct {
	ammoProvider
	size int
}

func (ap *LogAmmoProvider) Start() {
	go func() { // requests reader/generator
		for i := 0; i < ap.size; i++ {
			if a, err := ap.decoder.FromString(fmt.Sprintf(`{"message": "Job #%d"}`, i)); err == nil {
				ap.source <- a
			} else {
				log.Println("Error decoding log ammo: ", err)
			}
		}
		close(ap.source)
		log.Println("Ran out of ammo")
	}()
}

func NewLogAmmoProvider(size int) (ap AmmoProvider, err error) {
	ap = &LogAmmoProvider{
		size: size,
		ammoProvider: ammoProvider{
			decoder: &LogAmmoJsonDecoder{},
			source:  make(chan Ammo, 128),
		},
	}
	return
}

func NewLogGunFromConfig(c *GunConfig) (g Gun, err error) {
	return &LogGun{}, nil
}
