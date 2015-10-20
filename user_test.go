package main

import (
	"log"
	_ "testing"
	"time"
)

func ExampleUser() {

	pl := NewPeriodicLimiter(time.Second / 4)
	ap, _ := NewLogAmmoProvider(8)
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    make(chan Sample),
		limiter:    pl,
		done:       make(chan bool),
		gun:        &LogGun{},
	}
	go u.run()
	ap.Start()
	pl.Start()
	go func() {
		for r := range u.results {
			log.Println(r)
		}
	}()
	<-u.done

	log.Println("Done")
	// Output:
}
