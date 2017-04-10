// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package register

import (
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/plugin"
)

func RegisterPtr(ptr interface{}, name string, newPlugin interface{}, newDefaultConfigOptional ...interface{}) {
	plugin.Register(plugin.PtrType(ptr), name, newPlugin, newDefaultConfigOptional...)
}

func Provider(name string, newProvider interface{}, newDefaultConfigOptional ...interface{}) {
	var ptr *core.Provider
	RegisterPtr(ptr, name, newProvider, newDefaultConfigOptional...)
}

func Limiter(name string, newLimiter interface{}, newDefaultConfigOptional ...interface{}) {
	var ptr *core.Schedule
	RegisterPtr(ptr, name, newLimiter, newDefaultConfigOptional...)
}

func Gun(name string, newGun interface{}, newDefaultConfigOptional ...interface{}) {
	var ptr *core.Gun
	RegisterPtr(ptr, name, newGun, newDefaultConfigOptional...)
}

func Aggregator(name string, newResultListener interface{}, newDefaultConfigOptional ...interface{}) {
	var ptr *core.Aggregator
	RegisterPtr(ptr, name, newResultListener, newDefaultConfigOptional...)
}
