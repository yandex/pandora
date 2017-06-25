package engine

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/coreutil"
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
	Id              string
	Provider        core.Provider                 `config:"ammo"`
	Aggregator      core.Aggregator               `config:"result"`
	NewGun          func() (core.Gun, error)      `config:"gun"`
	RPSPerInstance  bool                          `config:"rps-per-instance"`
	NewRPSSchedule  func() (core.Schedule, error) `config:"rps"`
	StartupSchedule core.Schedule                 `config:"startup"`
}

type Instance struct {
	Id         string
	Provider   core.Provider
	Aggregator core.Aggregator
	Schedule   core.Schedule
	Gun        core.Gun
}

func (i *Instance) Start(ctx context.Context) error {
	// TODO(skipor): debug log about start and finish
	//log.Printf("Starting user: %s\n", i.Id)
	evUsersStarted.Add(1)
	defer func() {
		//log.Printf("Exit user: %s\n", i.Id)
		evUsersFinished.Add(1)
	}()
	i.Gun.Bind(i.Aggregator)
	nextShoot := coreutil.NewWaiter(i.Schedule, ctx)
	for {
		// Acquire should unblock in case of context cancel.
		ammo, more := i.Provider.Acquire()
		if !more {
			log.Println("Ammo ended")
			break
		}
		if !nextShoot.Wait() {
			break
		}
		evRequests.Add(1)
		i.Gun.Shoot(ctx, ammo)
		evResponses.Add(1)
		i.Provider.Release(ammo)
	}
	return ctx.Err()
}

func (p *InstancePool) Start(ctx context.Context) error {
	// TODO(skipor): info log about start and finish

	utilCtx, utilCancel := context.WithCancel(ctx)
	defer utilCancel()

	start := time.Now()
	p.StartupSchedule.Start(start)
	userPromises := utils.Promises{}
	utilsPromises := utils.Promises{
		utils.PromiseCtx(utilCtx, p.Provider.Start),
		utils.PromiseCtx(utilCtx, p.Aggregator.Start),
	}
	newInstanceSchedule := func() func() (core.Schedule, error) {
		if p.RPSPerInstance {
			return p.NewRPSSchedule
		}
		sharedSchedule, err := p.NewRPSSchedule()
		return func() (core.Schedule, error) {
			return sharedSchedule, err
		}
	}()
	startInstance := coreutil.NewWaiter(p.StartupSchedule, ctx)
	for startInstance.Wait() {
		shed, err := newInstanceSchedule()
		if err != nil {
			return fmt.Errorf("schedule create failed: %s", err)
		}
		gun, err := p.NewGun()
		if err != nil {
			return fmt.Errorf("gun create failed: %s", err)
		}
		// TODO(skipor): set unique user name
		u := &Instance{
			Id:         p.Id,
			Provider:   p.Provider,
			Aggregator: p.Aggregator,
			Schedule:   shed,
			Gun:        gun,
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
