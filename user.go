package main

type User struct {
	name       string
	ammunition chan Ammo
	results    chan Sample
	limiter    chan bool
	done       chan bool
	gun        Gun
}

type Sample interface {
	PhoutSample() *PhoutSample
	String() string
}

type Gun interface {
	Run(Ammo, chan<- Sample)
}

func (u *User) run() {
	for j := range u.ammunition {
		<-u.limiter
		u.gun.Run(j, u.results)
	}
	u.done <- true
}
