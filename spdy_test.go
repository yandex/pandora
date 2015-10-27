package main

import (
	"log"
	"testing"
	"time"
)

func ExampleSpdy() {
	ap, _ := NewHttpAmmoProvider("./example/data/ammo.jsonline")
	rl, _ := NewLoggingResultListener()
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    rl,
		limiter:    NewPeriodicLimiter(time.Second / 4),
		done:       make(chan bool),
		gun:        &SpdyGun{target: "localhost:3000"},
	}
	go u.run()
	u.ammunition.Start()
	u.results.Start()
	u.limiter.Start()
	<-u.done

	log.Println("Done")
	// Output:
}

func TestSpdyConfig(t *testing.T) {
	lc := &LimiterConfig{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    0.46,
			"BatchSize": 3.0,
			"MaxCount":  5.0,
		},
	}
	l, err := NewLimiterFromConfig(lc)
	if err != nil {
		t.Errorf("Error configuring limiter: %s", err)
	}
	apc := &AmmoProviderConfig{
		AmmoType:   "jsonline/spdy",
		AmmoSource: "./example/data/ammo.jsonline",
	}
	ap, err := NewAmmoProviderFromConfig(apc)
	if err != nil {
		t.Errorf("Error configuring ammo provider: %s", err)
	}
	rc := &ResultListenerConfig{
		ListenerType: "log/simple",
	}
	r, err := NewResultListenerFromConfig(rc)
	if err != nil {
		t.Errorf("Error configuring result listener: %s", err)
	}
	gc := &GunConfig{
		GunType: "spdy",
		Parameters: map[string]interface{}{
			"Target": "localhost:3000",
		},
	}
	g, err := NewGunFromConfig(gc)
	if err != nil {
		t.Errorf("Error configuring gun: %s", err)
	}
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    r,
		limiter:    l,
		done:       make(chan bool),
		gun:        g,
	}
	go u.run()
	u.ammunition.Start()
	u.results.Start()
	u.limiter.Start()
	<-u.done

	log.Println("Done")
	// Output:
}

func TestSpdyPhout(t *testing.T) {
	lc := &LimiterConfig{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    0.46,
			"BatchSize": 3.0,
			"MaxCount":  5.0,
		},
	}
	l, err := NewLimiterFromConfig(lc)
	if err != nil {
		t.Errorf("Error configuring limiter: %s", err)
		return
	}
	apc := &AmmoProviderConfig{
		AmmoType:   "jsonline/spdy",
		AmmoSource: "./example/data/ammo.jsonline",
	}
	ap, err := NewAmmoProviderFromConfig(apc)
	if err != nil {
		t.Errorf("Error configuring ammo provider: %s", err)
		return
	}
	rc := &ResultListenerConfig{
		ListenerType: "log/phout",
	}
	r, err := NewResultListenerFromConfig(rc)
	if err != nil {
		t.Errorf("Error configuring result listener: %s", err)
		return
	}
	gc := &GunConfig{
		GunType: "spdy",
		Parameters: map[string]interface{}{
			"Target": "localhost:3000",
		},
	}
	g, err := NewGunFromConfig(gc)
	if err != nil {
		t.Errorf("Error configuring gun: %s", err)
		return
	}
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    r,
		limiter:    l,
		done:       make(chan bool),
		gun:        g,
	}
	go u.run()
	u.ammunition.Start()
	u.results.Start()
	u.limiter.Start()
	<-u.done

	log.Println("Done")
	// Output:
}
