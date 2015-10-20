package main

type User struct {
	name       string
	ammunition AmmoProvider
	results    ResultListener
	limiter    Limiter
	done       chan bool
	gun        Gun
}

type Gun interface {
	Run(Ammo, chan<- Sample)
}

func (u *User) run() {
	control := u.limiter.Control()
	sink := u.results.Sink()
	for j := range u.ammunition.Source() {
		<-control
		u.gun.Run(j, sink)
	}
	u.done <- true
}
