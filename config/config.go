package config

import (
	"encoding/json"
)

type Global struct {
	Pools []UserPool
}

type AmmoProvider struct {
	AmmoType   string
	AmmoSource string // filename for ammo file (decoder based on ammo type)
	AmmoLimit  int    // limit of ammos, default is 0 - infinite ammos
	Passes     int    // number of passes, default is 0 - infinite passes
}

type Gun struct {
	GunType    string
	Parameters map[string]interface{}
}

type ResultListener struct {
	ListenerType string
	Destination  string
}

type Limiter struct {
	LimiterType string
	Parameters  map[string]interface{}
}

type CompositeLimiter struct {
	Steps []Limiter
}

type User struct {
	Name           string
	Gun            *Gun
	AmmoProvider   AmmoProvider
	ResultListener ResultListener
	Limiter        *Limiter
}

type UserPool struct {
	Name           string
	Gun            *Gun
	AmmoProvider   *AmmoProvider
	ResultListener *ResultListener
	UserLimiter    *Limiter
	StartupLimiter *Limiter
	SharedSchedule bool // wether or not will all Users from this pool have shared schedule
}

func NewGlobalFromJSON(jsonDoc []byte) (gc Global, err error) {
	err = json.Unmarshal(jsonDoc, &gc)
	return
}
