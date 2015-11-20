// +build !noHttpGun

package cmd

import (
	"github.com/yandex/pandora/extpoints"
	httpGun "github.com/yandex/pandora/gun/http"
)

func init() {
	// inject guns
	extpoints.Guns.Register(httpGun.New, "http")
}
