package engine

import (
	"log"
	"time"
)

func ExampleHttp() {
	ap, _ := NewHttpAmmoProvider("./example/data/ammo.jsonline", 10, 10)
	rl, _ := NewLoggingResultListener()
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    rl,
		limiter:    NewPeriodicLimiter(time.Second / 4),
		done:       make(chan bool),
		gun:        &HttpGun{target: "localhost:3000"},
	}
	go u.run()
	u.ammunition.Start()
	u.results.Start()
	u.limiter.Start()
	<-u.done

	log.Println("Done")
	// Output:
}
