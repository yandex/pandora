// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package core

import (
	"reflect"

	"github.com/spf13/afero"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregate"
	"github.com/yandex/pandora/core/aggregate/netsample"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/plugin"
	"github.com/yandex/pandora/core/plugin/pluginconfig"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/core/schedule"
)

func Import(fs afero.Fs) {
	register.Aggregator("phout", func(conf netsample.PhoutConfig) (core.Aggregator, error) {
		a, err := netsample.NewPhout(fs, conf)
		return netsample.WrapAggregator(a), err
	})

	register.Aggregator("discard", aggregate.NewDiscard)
	register.Aggregator("log", aggregate.NewLog)

	register.Limiter("line", schedule.NewLineConf)
	register.Limiter("const", schedule.NewConstConf)
	register.Limiter("once", schedule.NewOnceConf)
	register.Limiter("unlimited", schedule.NewUnlimitedConf)
	register.Limiter(compositeScheduleConfigName, schedule.NewCompositeConf)

	config.AddTypeHook(scheduleSliceToCompositeConfigHook)

	// Required for decoding plugins. Need to be added after Composite Schedule hacky hook.
	pluginconfig.AddHooks()
}

const compositeScheduleConfigName = "composite"

var scheduleType = plugin.PtrType((*core.Schedule)(nil))

// scheduleSliceToCompositeConfigHook helps to decode []interface{} as core.Schedule plugin.
func scheduleSliceToCompositeConfigHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.Slice {
		return data, nil
	}
	if t.Kind() != reflect.Interface && t.Kind() != reflect.Func {
		return data, nil
	}
	factoryPluginType, isPluginFactory := plugin.FactoryPluginType(t)
	isSchedule := t == scheduleType || isPluginFactory && factoryPluginType == scheduleType
	if !isSchedule {
		return data, nil
	}
	zap.L().Debug("Composite schedule hook triggered")
	return map[string]interface{}{
		pluginconfig.PluginNameKey: compositeScheduleConfigName,
		"nested":                   data,
	}, nil
}
