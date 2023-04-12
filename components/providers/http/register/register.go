package register

import (
	"github.com/yandex/pandora/components/providers/http/middleware"
	"github.com/yandex/pandora/core/register"
)

func HTTPMW(name string, mwConstructor interface{}, defaultConfigOptional ...interface{}) {
	var ptr *middleware.Middleware
	register.RegisterPtr(ptr, name, mwConstructor, defaultConfigOptional...)
}
