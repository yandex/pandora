package scenario

import (
	"net/http"

	"github.com/yandex/pandora/core/aggregator/netsample"
)

// TODO: Not used yet
type Ammo interface {
	// TODO(skipor): instead of sample use it wrapper with httptrace and more usable interface.
	Request() (*http.Request, *netsample.Sample)
	// Id unique ammo id. Usually equals to ammo num got from provider.
	ID() uint64
	IsInvalid() bool
}
