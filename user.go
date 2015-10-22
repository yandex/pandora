package main

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
	case "log":
		return NewLogGunFromConfig(c)
	default:
		err = errors.New(fmt.Sprintf("No such gun type: %s", c.GunType))
	}
	return
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
	userConfig        *UserConfig
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
		for range up.startupLimiter.Control() {
			u, err := NewUserFromConfig(up.userConfig)
			if err != nil {
				log.Fatal("could not make a user from config", err)
				break
			}
			u.ammunition = up.ammunition
			u.results = up.results
			u.run()
			up.users = append(up.users, u)
		}
		for _, u := range up.users {
			<-u.done
		}
		up.done <- true
	}()
}

func NewUserPoolFromConfig(c *UserPoolConfig) (u *UserPool, err error) {
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
	u = &UserPool{
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
