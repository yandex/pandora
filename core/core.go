package core

import (
	"context"
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

type Schedule interface {
	Start(context.Context) error
	Control() <-chan struct{}
}
