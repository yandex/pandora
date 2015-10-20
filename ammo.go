package main

type AmmoProvider interface {
	Start()
	Source() chan Ammo
}

type ammoProvider struct {
	decoder AmmoDecoder
	source  chan Ammo
}

func (ap *ammoProvider) Source() (s chan Ammo) {
	return ap.source
}

type AmmoDecoder interface {
	FromString(string) (Ammo, error)
}

type Ammo interface {
}
