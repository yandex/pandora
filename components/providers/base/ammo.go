package base

import "github.com/yandex/pandora/core/aggregator/netsample"

type Ammo[R any] struct {
	Req       *R
	tag       string
	id        uint64
	isInvalid bool
}

func (a *Ammo[R]) Request() (*R, *netsample.Sample) {
	sample := netsample.Acquire(a.Tag())
	sample.SetID(a.ID())
	return a.Req, sample
}

func (a *Ammo[R]) Reset(req *R, tag string) {
	a.Req = req
	a.tag = tag
	a.id = 0
	a.isInvalid = false
}

func (a *Ammo[_]) SetID(id uint64) {
	a.id = id
}

func (a *Ammo[_]) ID() uint64 {
	return a.id
}

func (a *Ammo[_]) Invalidate() {
	a.isInvalid = true
}

func (a *Ammo[_]) IsInvalid() bool {
	return a.isInvalid
}

func (a *Ammo[_]) IsValid() bool {
	return !a.isInvalid
}

func (a *Ammo[_]) Tag() string {
	return a.tag
}
