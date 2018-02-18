// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package core

import (
	"reflect"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/core/aggregator"
	"github.com/yandex/pandora/core/datasink"
	"github.com/yandex/pandora/lib/tag"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/plugin"
	"github.com/yandex/pandora/core/plugin/pluginconfig"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/core/schedule"
)

const (
	fileSinkKey          = "file"
	compositeScheduleKey = "composite"
)

func Import(fs afero.Fs) {

	register.DataSink(fileSinkKey, func(conf datasink.FileConfig) core.DataSink {
		return datasink.NewFile(fs, conf)
	})
	const (
		stdoutSinkKey = "stdout"
		stderrSinkKey = "stderr"
	)
	register.DataSink(stdoutSinkKey, datasink.NewStdout)
	register.DataSink(stderrSinkKey, datasink.NewStderr)
	AddSinkConfigHook(func(str string) (ok bool, pluginType string, _ map[string]interface{}) {
		for _, key := range []string{stdoutSinkKey, stderrSinkKey} {
			if str == key {
				return true, key, nil
			}
		}
		return
	})

	register.Aggregator("phout", func(conf netsample.PhoutConfig) (core.Aggregator, error) {
		a, err := netsample.NewPhout(fs, conf)
		return netsample.WrapAggregator(a), err
	})
	register.Aggregator("jsonlines", aggregator.NewJSONLinesAggregator, aggregator.DefaultJSONLinesAggregatorConfig)
	register.Aggregator("log", aggregator.NewLog)
	register.Aggregator("discard", aggregator.NewDiscard)

	register.Limiter("line", schedule.NewLineConf)
	register.Limiter("const", schedule.NewConstConf)
	register.Limiter("once", schedule.NewOnceConf)
	register.Limiter("unlimited", schedule.NewUnlimitedConf)
	register.Limiter(compositeScheduleKey, schedule.NewCompositeConf)

	config.AddTypeHook(sinkStringHook)
	config.AddTypeHook(scheduleSliceToCompositeConfigHook)

	// Required for decoding plugins. Need to be added after Composite Schedule hacky hook.
	pluginconfig.AddHooks()
}

var (
	scheduleType   = plugin.PtrType((*core.Schedule)(nil))
	dataSinkType   = plugin.PtrType((*core.DataSink)(nil))
	dataSourceType = plugin.PtrType((*core.DataSource)(nil))
)

func isPluginOrFactory(expectedPluginType, actualType reflect.Type) bool {
	if actualType.Kind() != reflect.Interface && actualType.Kind() != reflect.Func {
		return false
	}
	factoryPluginType, isPluginFactory := plugin.FactoryPluginType(actualType)
	return actualType == expectedPluginType || isPluginFactory && factoryPluginType == expectedPluginType
}

type PluginConfigStringHook func(str string) (ok bool, pluginType string, conf map[string]interface{})

var (
	dataSinkConfigHooks []PluginConfigStringHook
)

func AddSinkConfigHook(hook PluginConfigStringHook) {
	dataSinkConfigHooks = append(dataSinkConfigHooks, hook)
}

// sinkStringHook helps to decode string as core.DataSink plugin.
// Try use sink hooks and use file as fallback.
func sinkStringHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if !isPluginOrFactory(dataSinkType, t) {
		return data, nil
	}
	if tag.Debug {
		zap.L().Debug("DataSink string hook triggered")
	}
	var (
		ok         bool
		pluginType string
		conf       map[string]interface{}
	)
	dataStr := data.(string)

	for _, hook := range dataSinkConfigHooks {
		ok, pluginType, conf = hook(dataStr)
		if ok {
			break
		}
	}

	if !ok {
		pluginType = fileSinkKey
		conf = map[string]interface{}{
			"path": data,
		}
	}

	if conf == nil {
		conf = make(map[string]interface{})
	}
	conf[pluginconfig.PluginNameKey] = pluginType

	if tag.Debug {
		zap.L().Debug("Hooked DataSink config", zap.Any("config", conf))
	}
	return conf, nil
}

// scheduleSliceToCompositeConfigHook helps to decode []interface{} as core.Schedule plugin.
func scheduleSliceToCompositeConfigHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.Slice {
		return data, nil
	}
	if t.Kind() != reflect.Interface && t.Kind() != reflect.Func {
		return data, nil
	}
	if !isPluginOrFactory(scheduleType, t) {
		return data, nil
	}
	if tag.Debug {
		zap.L().Debug("Composite schedule hook triggered")
	}
	return map[string]interface{}{
		pluginconfig.PluginNameKey: compositeScheduleKey,
		"nested":                   data,
	}, nil
}
