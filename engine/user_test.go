package engine

import (
	"log"
	"testing"
	"time"
)

func ExampleUser() {

	pl := NewPeriodicLimiter(time.Second / 4)
	ap, _ := NewLogAmmoProvider(8)
	rl, _ := NewLoggingResultListener()
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    rl,
		limiter:    pl,
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

func TestUserPoolConfig(t *testing.T) {
	lc := &LimiterConfig{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    1.0,
			"BatchSize": 3.0,
			"MaxCount":  9.0,
		},
	}
	slc := &LimiterConfig{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    0.1,
			"BatchSize": 2.0,
			"MaxCount":  5.0,
		},
	}
	apc := &AmmoProviderConfig{
		AmmoType:   "jsonline/spdy",
		AmmoSource: "./testdata/ammo.jsonline",
	}
	rlc := &ResultListenerConfig{
		ListenerType: "log/simple",
	}
	gc := &GunConfig{
		GunType: "spdy",
		Parameters: map[string]interface{}{
			"Target": "localhost:3000",
		},
	}
	upc := &UserPoolConfig{
		Name:           "Pool#0",
		Gun:            gc,
		AmmoProvider:   apc,
		ResultListener: rlc,
		UserLimiter:    lc,
		StartupLimiter: slc,
	}
	up, err := NewUserPoolFromConfig(upc)
	if err != nil {
		t.Errorf("Could not create user pool: %s", err)
		return
	}
	up.Start()
	<-up.done

	log.Println("Done")
}
