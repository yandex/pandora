package config

import (
	"github.com/yandex/pandora/components/providers/http/middleware"
)

type Config struct {
	Decoder DecoderType
	File    string
	// Limit limits total num of ammo. Unlimited if zero.
	Limit uint
	// Default HTTP headers
	Headers []string
	// Passes limits ammo file passes. Unlimited if zero.
	// Only for `jsonline` decoder.
	Passes          uint
	Uris            []string
	ContinueOnError bool
	// Maximum number of byte in jsonline ammo. Default is bufio.MaxScanTokenSize
	MaxAmmoSize int
	ChosenCases []string
	Middlewares []middleware.Middleware
}
