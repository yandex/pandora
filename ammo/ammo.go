package ammo

import "context"

type Provider interface {
	Start(context.Context) error
	Source() <-chan Ammo
	// Release notifies that ammo usage is finished.
	Release(Ammo)
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
