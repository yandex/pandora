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
		results:    make(chan Sample),
		limiter:    NewPeriodicLimiter(time.Second / 4),
		done:       make(chan bool),
		gun:        &SpdyGun{target: "localhost:3000"},
	}
	go u.run()
	ap.Start()
	u.limiter.Start()
	go func() {
		for r := range u.results {
			log.Println(r)
		}
	}()
	<-u.done

	log.Println("Done")
	// Output:
}
