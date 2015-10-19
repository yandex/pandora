package main

type AmmoProvider interface {
	FromFile(string) <-chan Ammo
}

type Ammo interface {
	FromJson(string) error
}
