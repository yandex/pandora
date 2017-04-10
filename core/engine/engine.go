package engine

import (
	"context"
	"fmt"
	"log"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/utils"
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
	Pools []InstancePool
}

// TODO (skipor): test that mapstructure hooks are correctly called on array
// elements. If it is true make config.SetDefault(default interface{})
// where default is struct or struct factory, and use it to set default
// unique Id and SharedLimits = true.

type InstancePool struct {
	Id                 string
	Provider           core.Provider                 `config:"ammo"`
	Aggregator         core.Aggregator               `config:"result"`
	NewGun             func() (core.Gun, error)      `config:"gun"`
	IndividualSchedule bool                          `config:"individual-schedule"`
	StartupSchedule    core.Schedule                 `config:"startup-schedule"`
	NewRPSSchedule     func() (core.Schedule, error) `config:"rps-schedule"`
}

type Instance struct {
	Id         string
	Provider   core.Provider
	Aggregator core.Aggregator
	Limiter    core.Schedule
	Gun        core.Gun
}

func (u *Instance) Start(ctx context.Context) error {
	// TODO(skipor): debug log about start and finish
	//log.Printf("Starting user: %s\n", u.Id)
	evUsersStarted.Add(1)
	defer func() {
		//log.Printf("Exit user: %s\n", u.Id)
		evUsersFinished.Add(1)
	}()
	control := u.Limiter.Control()
	u.Gun.Bind(u.Aggregator)
	for ctx.Err() == nil {
		// Acquire should unblock in case of context cancel.
		ammo, more := u.Provider.Acquire()
		if !more {
			log.Println("Ammo ended")
			break
		}
		_, more = <-control
		if !more {
			log.Println("Schedule ended.")
			break
		}
		evRequests.Add(1)
		u.Gun.Shoot(ctx, ammo)
		evResponses.Add(1)
		u.Provider.Release(ammo)
	}
	return ctx.Err()
}

func (p *InstancePool) Start(ctx context.Context) error {
	// TODO(skipor): info log about start and finish

	// TODO(skipor): check that gun is compatible with ammo provider and
	// result listener before start, and print nice error message.

	utilCtx, utilCancel := context.WithCancel(ctx)
	defer utilCancel()

	userPromises := utils.Promises{}
	utilsPromises := utils.Promises{
		utils.PromiseCtx(utilCtx, p.Provider.Start),
		utils.PromiseCtx(utilCtx, p.Aggregator.Start),
		utils.PromiseCtx(utilCtx, p.StartupSchedule.Start),
	}
	var sharedSchedule core.Schedule

	if !p.IndividualSchedule {
		var err error
		sharedSchedule, err = p.NewRPSSchedule()
		if err != nil {
			return fmt.Errorf("could not make a user limiter from config due to %s", err)
		}
		// Starting shared schedule.
		// This may cause spike load in the beginning of a test if it takes time
		// to initialize a user, because we don't wait for them to initialize in
		// case of shared limiter and there might be some ticks accumulated
		utilsPromises = append(utilsPromises, utils.PromiseCtx(utilCtx, sharedSchedule.Start))
	}

	for range p.StartupSchedule.Control() {
		var l core.Schedule
		if !p.IndividualSchedule {
			l = sharedSchedule
		} else {
			var err error
			l, err = p.NewRPSSchedule()
			if err != nil {
				return fmt.Errorf("could not make a user limiter from config due to %s", err)
			}
		}
		g, err := p.NewGun()
		if err != nil {
			return fmt.Errorf("could not make a gun from config due to %s", err)
		}
		// TODO(skipor): set unique user name
		u := &Instance{
			Id:         p.Id,
			Provider:   p.Provider,
			Aggregator: p.Aggregator,
			Limiter:    l,
			Gun:        g,
		}
		if p.IndividualSchedule {
			utilsPromises = append(utilsPromises, utils.PromiseCtx(utilCtx, l.Start))
		}
		userPromises = append(userPromises, utils.PromiseCtx(ctx, u.Start))
	}
	// TODO(skipor): don't use promises, send all errors into only chan
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
