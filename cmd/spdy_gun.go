// +build !noSpdyGun

package cmd

import (
	"github.com/yandex/pandora/extpoints"
	spdyGun "github.com/yandex/pandora/gun/spdy"
)

func init() {
	// inject guns
	extpoints.Guns.Register(spdyGun.New, "spdy")
}
