package config

import (
	"encoding/json"
	"log"
	"testing"
)

func TestGlobalConfig(t *testing.T) {
	lc := &Limiter{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    1.0,
			"BatchSize": 3.0,
			"MaxCount":  9.0,
		},
	}
	slc := &Limiter{
		LimiterType: "periodic",
		Parameters: map[string]interface{}{
			"Period":    0.1,
			"BatchSize": 2.0,
			"MaxCount":  5.0,
		},
	}
	apc := &AmmoProvider{
		AmmoType:   "jsonline/spdy",
		AmmoSource: "./example/data/ammo.jsonline",
	}
	rlc := &ResultListener{
		ListenerType: "log/simple",
	}
	gc := &Gun{
		GunType: "spdy",
		Parameters: map[string]interface{}{
			"Target": "localhost:3000",
		},
	}
	globalConfig := &Global{
		Pools: []UserPool{
			{
				Name:           "Pool#0",
				Gun:            gc,
				AmmoProvider:   apc,
				ResultListener: rlc,
				UserLimiter:    lc,
				StartupLimiter: slc,
			},
			{
				Name:           "Pool#1",
				Gun:            gc,
				AmmoProvider:   apc,
				ResultListener: rlc,
				UserLimiter:    lc,
				StartupLimiter: slc,
			},
		},
	}
	jsonDoc, err := json.Marshal(globalConfig)
	if err != nil {
		t.Errorf("Could not marshal config to json: %s", err)
		return
	}

	log.Println(string(jsonDoc))

	newgc, err := NewGlobalFromJSON(jsonDoc)
	if err != nil {
		t.Errorf("Could not unmarshal config from json: %s", err)
		return
	}
	log.Printf("\n%#v\n", newgc)
}
