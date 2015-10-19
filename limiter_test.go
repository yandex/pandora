package main

import (
	"fmt"
	"log"
	_ "testing"
	"time"
)

func ExampleBatchLimiter() {
	bl := NewBatchLimiter(10, NewPeriodicLimiter(time.Second))
	u := &User{
		name:       "Example user",
		ammunition: make(chan Ammo, 10),
		results:    make(chan Sample),
		limiter:    bl,
		done:       make(chan bool),
		gun:        &LogGun{},
	}
	go u.run()
	go func() {
		for i := 0; i < 50; i++ {
			u.ammunition <- &LogAmmo{fmt.Sprintf("{'message': 'Job #%d'}", i)}
		}
		close(u.ammunition)
	}()
	bl.Start()
	go func() {
		for r := range u.results {
			log.Println(r)
		}
	}()
	<-u.done

	log.Println("Done")
	// Output:
}
