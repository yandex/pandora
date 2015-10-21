package main

import (
	"log"
	_ "testing"
	"time"
)

func ExampleSpdy() {
	ap, _ := NewHttpAmmoProvider("./ammo.jsonline")
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    NewLoggingResultListener(),
		limiter:    NewPeriodicLimiter(time.Second / 4),
		done:       make(chan bool),
		gun:        &SpdyGun{target: "localhost:3000"},
	}
	go u.run()
	u.ammunition.Start()
	u.results.Start()
	u.limiter.Start()
	<-u.done

	log.Println("Done")
	// Output:
}

func ExampleSpdyConfig() {
	lc := &LimiterConfig{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    0.46,
			"BatchSize": 3.0,
			"MaxCount":  5.0,
		},
	}
	l, _ := NewLimiterFromConfig(lc)
	apc := &AmmoProviderConfig{
		AmmoType:   "dummy",
		AmmoSource: "./ammo.jsonline",
	}
	ap, _ := NewHttpAmmoProvider("./ammo.jsonline")
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    NewLoggingResultListener(),
		limiter:    NewPeriodicLimiter(time.Second / 4),
		done:       make(chan bool),
		gun:        &SpdyGun{target: "localhost:3000"},
	}
	go u.run()
	u.ammunition.Start()
	u.results.Start()
	u.limiter.Start()
	<-u.done

	log.Println("Done")
	// Output:
}
