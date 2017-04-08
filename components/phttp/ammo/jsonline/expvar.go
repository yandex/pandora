package jsonline

import (
	"log"
	"time"

	"github.com/yandex/pandora/lib/monitoring"
)

var evPassesLeft = monitoring.NewCounter("ammo_PassesLeft")

// TODO: use one rcrowley/go-metrics Registry and print metrics from it.
func init() {
	go func() {
		passesLeft := evPassesLeft.Get()
		for range time.NewTicker(1 * time.Second).C {
			if passesLeft < 0 {
				return // infinite number of passes
			}
			newPassesLeft := evPassesLeft.Get()
			if newPassesLeft != passesLeft {
				log.Printf("[AMMO] passes left: %d", newPassesLeft)
				passesLeft = newPassesLeft
			}
		}
	}()
}
