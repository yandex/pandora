package engine

import (
	"errors"
	"fmt"
	"log"
)

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

func NewGunFromConfig(c *GunConfig) (g Gun, err error) {
	if c == nil {
		return
	}
	switch c.GunType {
	case "spdy":
		return NewSpdyGunFromConfig(c)
	case "http":
		return NewHttpGunFromConfig(c)
	case "log":
		return NewLogGunFromConfig(c)
	default:
		err = errors.New(fmt.Sprintf("No such gun type: %s", c.GunType))
	}
	return
}

func (u *User) run() {
	log.Printf("Starting user: %s\n", u.name)
	defer func() {
		log.Printf("Exit user: %s\n", u.name)
		u.done <- true
	}()
	control := u.limiter.Control()
	source := u.ammunition.Source()
	sink := u.results.Sink()
	for {
		j, more := <-source
		if !more {
			log.Println("Ammo ended")
			return
		}
		_, more = <-control
		if more {
			u.gun.Run(j, sink)
		} else {
			log.Println("Limiter ended.")
			return
		}
	}
}

func NewUserFromConfig(c *UserConfig) (u *User, err error) {
	if c == nil {
		return
	}
	gun, err := NewGunFromConfig(c.Gun)
	if err != nil {
		return nil, err
	}
	ammunition, err := NewAmmoProviderFromConfig(c.AmmoProvider)
	if err != nil {
		return nil, err
	}
	results, err := NewResultListenerFromConfig(c.ResultListener)
	if err != nil {
		return nil, err
	}
	limiter, err := NewLimiterFromConfig(c.Limiter)
	if err != nil {
		return nil, err
	}
	u = &User{
		name:       c.Name,
		ammunition: ammunition,
		results:    results,
		limiter:    limiter,
		done:       make(chan bool, 1),
		gun:        gun,
	}
	return
}

type UserPool struct {
	name              string
	userLimiterConfig *LimiterConfig
	gunConfig         *GunConfig
	ammunition        AmmoProvider
	results           ResultListener
	startupLimiter    Limiter
	users             []*User
	done              chan bool
}

func (up *UserPool) Start() {
	up.users = make([]*User, 0, 128)
	go func() {
		i := 0
		for range up.startupLimiter.Control() {
			l, err := NewLimiterFromConfig(up.userLimiterConfig)
			if err != nil {
				log.Fatal("could not make a user limiter from config", err)
				break
			}
			g, err := NewGunFromConfig(up.gunConfig)
			if err != nil {
				log.Fatal("could not make a gun from config", err)
				break
			}
			u := &User{
				name:       fmt.Sprintf("%s/%d", up.name, i),
				ammunition: up.ammunition,
				results:    up.results,
				limiter:    l,
				done:       make(chan bool),
				gun:        g,
			}
			l.Start()
			go u.run()
			up.users = append(up.users, u)
			i += 1
		}
		log.Println("Started all users. Waiting for them")
		for _, u := range up.users {
			<-u.done
		}
		up.done <- true
	}()
	up.ammunition.Start()
	up.results.Start()
	up.startupLimiter.Start()
}

func NewUserPoolFromConfig(c *UserPoolConfig) (up *UserPool, err error) {
	if c == nil {
		return nil, errors.New("no pool config provided")
	}
	ammunition, err := NewAmmoProviderFromConfig(c.AmmoProvider)
	if err != nil {
		return nil, err
	}
	results, err := NewResultListenerFromConfig(c.ResultListener)
	if err != nil {
		return nil, err
	}
	startupLimiter, err := NewLimiterFromConfig(c.StartupLimiter)
	if err != nil {
		return nil, err
	}
	up = &UserPool{
		name:              c.Name,
		ammunition:        ammunition,
		results:           results,
		startupLimiter:    startupLimiter,
		done:              make(chan bool, 1),
		gunConfig:         c.Gun,
		userLimiterConfig: c.UserLimiter,
	}
	return
}
