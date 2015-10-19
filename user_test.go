package main

import (
	"fmt"
	"log"
	_ "testing"
	"time"
)

func ExampleUser() {

	pl := NewPeriodicLimiter(time.Second / 4)
	u := &User{
		name:       "Example user",
		ammunition: make(chan Ammo, 10),
		results:    make(chan Sample),
		limiter:    pl.Control(),
		done:       make(chan bool),
		gun:        &LogGun{},
	}
	go u.run()
	for i := 0; i < 5; i++ {
		u.ammunition <- &LogAmmo{fmt.Sprintf("{'message': 'Job #%d'}", i)}
	}
	close(u.ammunition)
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
