package ammo

import (
	"context"
	"reflect"
	"sync"

	"github.com/yandex/pandora/plugin"
)

type Provider interface {
	Start(context.Context) error
	Source() <-chan Ammo
	Release(Ammo) // return unused Ammo object to memory pool
}

func RegisterProvider(name string, newAmmoProvider interface{}, newDefaultConfigOptional ...interface{}) {
	plugin.Register(reflect.TypeOf((*Provider)(nil)).Elem(), name, newAmmoProvider, newDefaultConfigOptional...)
}

type BaseProvider struct {
	decoder Decoder
	source  <-chan Ammo
	pool    sync.Pool
}

type Ammo interface{}

type Decoder interface {
	Decode([]byte, Ammo) (Ammo, error)
}

func NewBaseProvider(source <-chan Ammo, decoder Decoder, New func() interface{}) *BaseProvider {
	return &BaseProvider{
		source:  source,
		decoder: decoder,
		pool:    sync.Pool{New: New},
	}
}

func (ap *BaseProvider) Source() <-chan Ammo {
	return ap.source
}

func (ap *BaseProvider) Release(a Ammo) {
	ap.pool.Put(a)
}

func (ap *BaseProvider) decode(src []byte) (Ammo, error) {
	a := ap.pool.Get()
	return ap.decoder.Decode(src, a)
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
