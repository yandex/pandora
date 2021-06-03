// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package register

import (
	"a.yandex-team.ru/load/projects/pandora/core"
	"a.yandex-team.ru/load/projects/pandora/core/plugin"
)

func RegisterPtr(ptr interface{}, name string, newPlugin interface{}, defaultConfigOptional ...interface{}) {
	plugin.Register(plugin.PtrType(ptr), name, newPlugin, defaultConfigOptional...)
}

func Provider(name string, newProvider interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.Provider
	RegisterPtr(ptr, name, newProvider, defaultConfigOptional...)
}

func Limiter(name string, newLimiter interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.Schedule
	RegisterPtr(ptr, name, newLimiter, defaultConfigOptional...)
}

func Gun(name string, newGun interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.Gun
	RegisterPtr(ptr, name, newGun, defaultConfigOptional...)
}

func Aggregator(name string, newAggregator interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.Aggregator
	RegisterPtr(ptr, name, newAggregator, defaultConfigOptional...)
}

func DataSource(name string, newDataSource interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.DataSource
	RegisterPtr(ptr, name, newDataSource, defaultConfigOptional...)
}

func DataSink(name string, newDataSink interface{}, defaultConfigOptional ...interface{}) {
	var ptr *core.DataSink
	RegisterPtr(ptr, name, newDataSink, defaultConfigOptional...)
}
