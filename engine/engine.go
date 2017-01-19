package engine

import (
	"context"
	"fmt"
	"log"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo"
	"github.com/yandex/pandora/gun"
	"github.com/yandex/pandora/limiter"
	"github.com/yandex/pandora/utils"
)

type Engine struct {
	config Config
}

func New(conf Config) *Engine {
	return &Engine{conf}
}

func (e *Engine) Serve(ctx context.Context) error {
	promises := utils.Promises{}
	for _, up := range e.config.Pools {
		promises = append(promises, utils.PromiseCtx(ctx, up.Start))
	}
	select {
	case err := <-promises.All():
		if err != nil {
			return err
		}
	case <-ctx.Done():
	}
	log.Println("Done")
	return nil
}

type Config struct {
	Pools []UserPool
}

// TODO (skipor): test that mapstructure hooks are correctly called on array
// elements. If it is true make config.SetDefault(default interface{})
// where default is struct or struct factory, and use it to set default
// unique Name and SharedLimits = true.

type UserPool struct {
	Name           string
	AmmoProvider   ammo.Provider                   `config:"ammo"`
	ResultListener aggregate.ResultListener        `config:"result"`
	SharedLimits   bool                            `config:"shared-limits"`
	StartupLimiter limiter.Limiter                 `config:"startup-limiter"`
	NewUserLimiter func() (limiter.Limiter, error) `config:"user-limiter"`
	NewGun         func() (gun.Gun, error)         `config:"gun"`
}

type User struct {
	Name       string
	Ammunition ammo.Provider
	Results    aggregate.ResultListener
	Limiter    limiter.Limiter
	Gun        gun.Gun
}

func (u *User) Start(ctx context.Context) error {
	//log.Printf("Starting user: %s\n", u.Name)
	evUsersStarted.Add(1)
	defer func() {
		//log.Printf("Exit user: %s\n", u.Name)
		evUsersFinished.Add(1)
	}()
	control := u.Limiter.Control()
	source := u.Ammunition.Source()
	u.Gun.BindResultsTo(u.Results.Sink())
loop:
	for {
		select {
		case ammo, more := <-source:
			if !more {
				log.Println("Ammo ended")
				break loop
			}
			_, more = <-control
			if more {
				evRequests.Add(1)
				u.Gun.Shoot(ctx, ammo)
				evResponses.Add(1)
				u.Ammunition.Release(ammo)
			} else {
				//log.Println("Limiter ended.")
				break loop
			}
		case <-ctx.Done():
			break loop
		}
	}
	return nil
}

func (p *UserPool) Start(ctx context.Context) error {
	//TODO
	// userCtx will be canceled when all users finished their execution

	utilCtx, utilCancel := context.WithCancel(ctx)
	defer utilCancel()

	userPromises := utils.Promises{}
	utilsPromises := utils.Promises{
		utils.PromiseCtx(utilCtx, p.AmmoProvider.Start),
		utils.PromiseCtx(utilCtx, p.ResultListener.Start),
		utils.PromiseCtx(utilCtx, p.StartupLimiter.Start),
	}
	var sharedLimiter limiter.Limiter

	if p.SharedLimits {
		var err error
		sharedLimiter, err = p.NewUserLimiter()
		if err != nil {
			return fmt.Errorf("could not make a user limiter from config due to %s", err)
		}
		// Starting shared limiter.
		// This may cause spike load in the beginning of a test if it takes time
		// to initialize a user, because we don't wait for them to initialize in
		// case of shared limiter and there might be some ticks accumulated
		utilsPromises = append(utilsPromises, utils.PromiseCtx(utilCtx, sharedLimiter.Start))
	}

	for range p.StartupLimiter.Control() {
		var l limiter.Limiter
		if p.SharedLimits {
			l = sharedLimiter
		} else {
			var err error
			l, err = p.NewUserLimiter()
			if err != nil {
				return fmt.Errorf("could not make a user limiter from config due to %s", err)
			}
		}
		g, err := p.NewGun()
		if err != nil {
			return fmt.Errorf("could not make a gun from config due to %s", err)
		}
		// TODO: set unique user name
		u := &User{
			Name:       p.Name,
			Ammunition: p.AmmoProvider,
			Results:    p.ResultListener,
			Limiter:    l,
			Gun:        g,
		}
		if !p.SharedLimits {
			utilsPromises = append(utilsPromises, utils.PromiseCtx(utilCtx, l.Start))
		}
		userPromises = append(userPromises, utils.PromiseCtx(ctx, u.Start))
	}
	// TODO: don't use promises, send all errors into only chan
	log.Println("Started all users. Waiting for them")
	err := <-userPromises.All()
	log.Println("Stop utils")
	utilCancel() // stop result listener when all users finished

	err2 := <-utilsPromises.All()
	if err2 != nil {
		fmt.Printf("Error waiting utils promises: %s", err2.Error())
	}
	return err
}
