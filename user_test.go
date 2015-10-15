package main

import (
	"fmt"
	"log"
	_ "testing"
	"time"
)

func ExampleUser() {
	u := &User{
		name:    "Example user",
		jobs:    make(chan Job, 10),
		results: make(chan Sample),
		limiter: make(chan bool),
		done:    make(chan bool),
		gun:     &LogGun{},
	}
	go u.run()
	for i := 0; i < 5; i++ {
		u.jobs <- &LogJob{fmt.Sprintf("Job #%d", i)}
	}
	close(u.jobs)
	go func() {
		for range time.NewTicker(time.Second).C {
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
