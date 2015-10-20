package main

import (
	"log"
	_ "testing"
	"time"
)

func ExampleBatchLimiter() {
	ap, _ := NewLogAmmoProvider(30)
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    NewLoggingResultListener(),
		limiter:    NewBatchLimiter(10, NewPeriodicLimiter(time.Second)),
		done:       make(chan bool),
		gun:        &LogGun{},
	}
	go u.run()
	u.ammunition.Start()
	u.results.Start()
	u.limiter.Start()
	<-u.done

	log.Println("Done")
	// Output:
}
