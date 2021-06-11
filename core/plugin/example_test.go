// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin_test

import "a.yandex-team.ru/load/projects/pandora/core/plugin"

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
