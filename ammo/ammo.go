package ammo

import "context"

type Provider interface {
	Start(context.Context) error
	Source() <-chan Ammo
	Release(Ammo) // return unused Ammo object to memory pool
}

type Ammo interface{}

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
