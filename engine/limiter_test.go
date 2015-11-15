package engine

import (
	"log"
	"testing"
	"time"
)

func ExampleBatchLimiter() {
	ap, _ := NewLogAmmoProvider(30)
	rl, _ := NewLoggingResultListener()
	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    rl,
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

func TestEmptyPeriodicLimiterConfig(t *testing.T) {
	lc := &LimiterConfig{
		LimiterType: "periodic",
		Parameters:  nil,
	}
	l, err := NewLimiterFromConfig(lc)

	if err == nil {
		t.Errorf("Should return error if empty config")
	}
	if l != nil {
		t.Errorf("Should return 'nil' if empty config")
	}
}

func TestPeriodicLimiterNoBatch(t *testing.T) {
	lc := &LimiterConfig{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period": 0.46,
		},
	}
	l, err := NewLimiterFromConfig(lc)

	if err != nil {
		t.Errorf("Got an error while creating periodic limiter: %s", err)
	}
	if l == nil {
		t.Errorf("Returned 'nil' with valid config")
	}
	switch tt := l.(type) {
	case *periodicLimiter:
	default:
		t.Errorf("Wrong limiter type returned (expected periodicLimiter): %T", tt)
	}
}

func TestPeriodicLimiterBatch(t *testing.T) {
	lc := &LimiterConfig{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    0.46,
			"BatchSize": 3.0,
		},
	}
	l, err := NewLimiterFromConfig(lc)

	if err != nil {
		t.Errorf("Got an error while creating periodic limiter: %s", err)
	}
	if l == nil {
		t.Errorf("Returned 'nil' with valid config")
	}
	switch tt := l.(type) {
	case *batchLimiter:
	default:
		t.Errorf("Wrong limiter type returned (expected batchLimiter): %T", tt)
	}
}

func TestPeriodicLimiterBatchMaxCount(t *testing.T) {
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
		t.Errorf("Got an error while creating periodic limiter: %s", err)
	}
	if l == nil {
		t.Errorf("Returned 'nil' with valid config")
	}
	switch tt := l.(type) {
	case *batchLimiter:
	default:
		t.Errorf("Wrong limiter type returned (expected batchLimiter): %T", tt)
	}
	l.Start()
	for range l.Control() {
		log.Println("Next tick")
	}
}

func ExampleBatchLimiterConfig() {
	ap, _ := NewLogAmmoProvider(30)
	lc := &LimiterConfig{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    0.46,
			"BatchSize": 3.0,
			"MaxCount":  5.0,
		},
	}
	l, _ := NewLimiterFromConfig(lc)
	rl, _ := NewLoggingResultListener()

	u := &User{
		name:       "Example user",
		ammunition: ap,
		results:    rl,
		limiter:    l,
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
