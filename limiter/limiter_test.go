package limiter

//func ExampleBatchLimiter() {
//	ap, _ := NewLogAmmoProvider(30)
//	rl, _ := NewLoggingResultListener()
//	u := &User{
//		name:       "Example user",
//		ammunition: ap,
//		results:    rl,
//		limiter:    NewBatch(10, NewPeriodic(time.Second)),
//		done:       make(chan bool),
//		gun:        &LogGun{},
//	}
//	go u.run()
//	u.ammunition.Start()
//	u.results.Start()
//	u.limiter.Start()
//	<-u.done
//
//	log.Println("Done")
//	// Output:
//}

//func TestEmptyPeriodicLimiterConfig(t *testing.T) {
//	lc := &config.Limiter{
//		LimiterType: "periodic",
//		Parameters:  nil,
//	}
//	l, err := NewPeriodicFromConfig(lc)
//
//	if err == nil {
//		t.Errorf("Should return error if empty config")
//	}
//	if l != nil {
//		t.Errorf("Should return 'nil' if empty config")
//	}
//}

//func ExampleBatchLimiterConfig() {
//	ap, err := ammo.NewLogAmmoProvider(&config.AmmoProvider{
//		AmmoLimit: 30,
//	})
//	if err != nil {
//		log.Panic(err)
//	}
//
//	lc := &config.Limiter{
//		LimiterType: "periodic",
//		Parameters: map[string]interface{}{
//			"Period":    0.46,
//			"BatchSize": 3.0,
//			"MaxCount":  5.0,
//		},
//	}
//	l, err := NewPeriodicFromConfig(lc)
//	if err != nil {
//		log.Panic(err)
//	}
//
//	rl, err := aggregate.NewLoggingResultListener(nil)
//	if err != nil {
//		log.Panic(err)
//	}
//
//	u := &engine.User{
//		Name:       "Example user",
//		Ammunition: ap,
//		Results:    rl,
//		Limiter:    l,
//		Done:       make(chan bool),
//		Gun:        gun.LogGun{},
//	}
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
//	defer cancel()
//
//	go u.Run(ctx)
//	u.Ammunition.Start(ctx)
//	u.Results.Start(ctx)
//	u.Limiter.Start(ctx)
//	<-u.Done
//
//	log.Println("Done")
//	// Output:
//}
