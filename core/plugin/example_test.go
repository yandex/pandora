package plugin_test

import "github.com/yandex/pandora/core/plugin"

type Plugin interface {
	DoSmth()
}

func RegisterPlugin(name string, newPluginImpl interface{}, newDefaultConfigOptional ...interface{}) {
	var p Plugin
	plugin.Register(plugin.PtrType(&p), name, newPluginImpl, newDefaultConfigOptional...)
}

func ExampleRegister() {
	type Conf struct{ Smth string }
	New := func(Conf) Plugin { panic("") }
	RegisterPlugin("no-default", New)
	RegisterPlugin("default", New, func() Conf { return Conf{"example"} })
}
