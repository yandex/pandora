package main

import (
	"log"
	"testing"
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

func TestNotExistentLimiterConfig(t *testing.T) {
	lc := &LimiterConfig{
		LimiterType: "NOT_EXISTENT",
		Parameters:  nil,
	}
	l, err := NewLimiterFromConfig(lc)

	if err == nil {
		t.Errorf("No error on non existent limiter type")
	}
	if l != nil {
		t.Errorf("Returned non-nil limiter for non existent limiter type")
	}
}

func TestEmptyLimiterConfig(t *testing.T) {
	l, err := NewLimiterFromConfig(nil)

	if err != nil {
		t.Errorf("Error on empty limiter config: %s", err)
	}
	if l != nil {
		t.Errorf("Returned non-nil limiter for empty config")
	}
}

func TestLimiterTypes(t *testing.T) {
	limiterTypes := []string{
		"periodic",
	}
	for _, limiterType := range limiterTypes {
		lc := &LimiterConfig{
			LimiterType: limiterType,
			Parameters:  nil,
		}
		l, err := NewLimiterFromConfig(lc)

		if err != nil {
			t.Errorf("Got an error while creating limiter of type '%s': %s", limiterType, err)
		}
		if l == nil {
			t.Errorf("Returned 'nil' as limiter of type: %s", limiterType)
		}
	}
}
