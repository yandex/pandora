package main

type User struct {
	name    string
	jobs    chan Job
	results chan Sample
	limiter chan bool
	done    chan bool
	gun     Gun
}

type Sample interface {
	PhoutSample() *PhoutSample
	String() string
}

type Gun interface {
	Run(Job, chan<- Sample)
}

type Job interface{}

func (u *User) run() {
	for j := range u.jobs {
		<-u.limiter
		u.gun.Run(j, u.results)
	}
	u.done <- true
}
