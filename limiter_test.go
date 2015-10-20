package main

import (
	"log"
	_ "testing"
	"time"
)

func ExampleBatchLimiter() {
	bl := NewBatchLimiter(10, NewPeriodicLimiter(time.Second))
	ap, _ := NewLogAmmoProvider(30)
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    make(chan Sample),
		limiter:    bl,
		done:       make(chan bool),
		gun:        &LogGun{},
	}
	go u.run()
	ap.Start()
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
