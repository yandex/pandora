package core

import (
	"context"
	"time"
)

// TODO(skipor): doc

type Ammo interface{}

//go:generate mockery -name=Provider -case=underscore -outpkg=coremock

type Provider interface {
	Start(context.Context) error
	// Acquire acquires ammo for shoot.
	// Ok false means that shooting should be stopped: ammo finished or shooting is canceled.
	Acquire() (a Ammo, ok bool)
	// Release notifies that ammo usage is finished.
	Release(Ammo)
}

type Sample interface{}

//go:generate mockery -name=Aggregator -case=underscore -outpkg=coremock

type Aggregator interface {
	Start(context.Context) error
	Release(Sample)
}

//go:generate mockery -name=Gun -case=underscore -outpkg=coremock

type Gun interface {
	Shoot(context.Context, Ammo) error
	Bind(Aggregator)
}

//go:generate mockery -name=Schedule -case=underscore -outpkg=coremock

// Schedule represents started operation schedule. Schedule is goroutine safe.
type Schedule interface {
	// Start starts schedule at passed time.
	// Start may be called once, before any Next call.
	// If start is not called, schedule started at first Next call.
	Start(startAt time.Time)
	// Next withdraw one operation token and returns next operation time and
	// ok equal true, when schedule is not finished.
	// If there is no operation tokens left, Next returns Schedule
	// finish time and ok equals false.
	Next() (ts time.Time, ok bool)
}
