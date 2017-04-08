package core

import (
	"context"

	"github.com/yandex/pandora/core/aggregate"
)

// Limiter interface describes limiter control structure
type Limiter interface {
	Start(context.Context) error
	Control() <-chan struct{}
}

type Provider interface {
	Start(context.Context) error
	Source() <-chan Ammo
	// Release notifies that ammo usage is finished.
	Release(Ammo)
}

type Ammo interface{}

type Gun interface {
	Shoot(context.Context, Ammo) error
	BindResultsTo(Results)
}

type Results chan<- *aggregate.Sample

func NewResults(buf int) chan *aggregate.Sample {
	return make(chan *aggregate.Sample, buf)
}

// Drain reads all ammos from ammo.Provider. Useful for tests.
func Drain(ctx context.Context, p Provider) []Ammo {
	ammos := []Ammo{}
loop:
	for {
		select {
		case a, more := <-p.Source():
			if !more {
				break loop
			}
			ammos = append(ammos, a)
		case <-ctx.Done():
			break loop
		}
	}
	return ammos
}
