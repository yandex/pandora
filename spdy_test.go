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
