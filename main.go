//go:generate go-extpoints
package main

import (
	"github.com/yandex/pandora/cmd"
)

// If you want to extend pandora with another guns
// Just copy this main.go and import them.
// They should connect to extpoints automatically

func main() {
	cmd.Run()
}
