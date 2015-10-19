package main

import (
	"log"
	_ "testing"
	"time"
)

func ExampleSpdy() {

	ammoJson := `{"uri": "/mapkit2/layers/2.x/map/tiles?scale=2&v=4.31.0&x=40780&y=20362&z=16&deviceid=0&lang=ru_RU&miid=aeb0c532a4de86e649f3383f543608c038d194053&test_buckets=11231%2C43%2C43&uuid=0", "method": "GET", "headers": {"Accept": "*/*", "Accept-Encoding": "gzip, deflate", "X-YRuntime-Signature": "ddc184bf55a26a19151aaa71b38c40f22f84fd8f", "Host": "spdy3.mob.maps.yandex.net", "User-Agent": "ru.yandex.yandexmaps.dev/1.0 mapkit/2.0 android/4.4.2 (samsung; GT-N7100; ru_RU)"}, "host": "spdy3.mob.maps.yandex.net"}`
	u := &User{
		name:       "Example user",
		ammunition: make(chan Ammo, 10),
		results:    make(chan Sample),
		limiter:    NewPeriodicLimiter(time.Second / 4),
		done:       make(chan bool),
		gun:        &SpdyGun{target: "localhost:3000"},
	}
	go u.run()
	for i := 0; i < 5; i++ {
		a := &HttpAmmo{}
		a.FromJson(ammoJson)
		u.ammunition <- a
	}
	close(u.ammunition)
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
