package ammo

import "golang.org/x/net/context"

type Provider interface {
	Start(context.Context) error
	Source() <-chan Ammo
}

type BaseProvider struct {
	decoder Decoder
	source  <-chan Ammo
}

type Ammo interface{}

type Decoder func([]byte) (Ammo, error)

func NewBaseProvider(source <-chan Ammo, decoder Decoder) *BaseProvider {
	return &BaseProvider{
		source:  source,
		decoder: decoder,
	}
}

func (ap *BaseProvider) Source() <-chan Ammo {
	return ap.source
}

func (ap *BaseProvider) Decode(src []byte) (Ammo, error) {
	return ap.decoder(src)
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
