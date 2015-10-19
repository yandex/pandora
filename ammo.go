package main

type AmmoProvider interface {
	FromFile(string) <-chan Ammo
}

type AmmoDecoder interface {
	FromString(string) (Ammo, error)
}

type Ammo interface {
}
