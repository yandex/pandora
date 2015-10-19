package main

import (
	"fmt"
	"log"
	_ "testing"
	"time"
)

func ExampleUser() {
	u := &User{
		name:       "Example user",
		ammunition: make(chan Ammo, 10),
		results:    make(chan Sample),
		limiter:    make(chan bool),
		done:       make(chan bool),
		gun:        &LogGun{},
	}
	go u.run()
	for i := 0; i < 5; i++ {
		u.ammunition <- &LogAmmo{fmt.Sprintf("{'message': 'Job #%d'}", i)}
	}
	close(u.ammunition)
	go func() {
		for range time.NewTicker(time.Second / 4).C {
			u.limiter <- true
		}
	}()
	go func() {
		for r := range u.results {
			log.Println(r)
		}
	}()
	<-u.done

	log.Println("Done")
	// Output:
}
