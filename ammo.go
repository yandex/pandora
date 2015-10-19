package main

type AmmoProvider interface {
	Start()
	Source() chan Ammo
}

type AmmoDecoder interface {
	FromString(string) (Ammo, error)
}

type Ammo interface {
}
