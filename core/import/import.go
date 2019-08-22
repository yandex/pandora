// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coreimport

import (
	"reflect"

	"github.com/spf13/afero"
	"go.uber.org/zap"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/core/datasink"
	"github.com/yandex/pandora/core/datasource"
	"github.com/yandex/pandora/core/plugin"
	"github.com/yandex/pandora/core/plugin/pluginconfig"
	"github.com/yandex/pandora/core/provider"
	"github.com/yandex/pandora/core/register"
	"github.com/yandex/pandora/core/schedule"
	"github.com/yandex/pandora/lib/tag"
)

const (
	fileDataKey          = "file"
	compositeScheduleKey = "composite"
)
//getter for fs to avoid afero dependency in custom guns
func GetFs() afero.Fs {
        return afero.NewOsFs()
}

func Import(fs afero.Fs) {

	register.DataSink(fileDataKey, func(conf datasink.FileConfig) core.DataSink {
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

	register.DataSource(fileDataKey, func(conf datasource.FileConfig) core.DataSource {
		return datasource.NewFile(fs, conf)
	})
	const (
		stdinSourceKey = "stdin"
	)
	register.DataSource(stdinSourceKey, datasource.NewStdin)
	AddSinkConfigHook(func(str string) (ok bool, pluginType string, _ map[string]interface{}) {
		if str != stdinSourceKey {
			return
		}
		return true, stdinSourceKey, nil
	})
	register.DataSource("inline", datasource.NewInline)

	// NOTE(skipor): json provider SHOULD NOT used normally. Register your own, that will return
	// type that you need, but untyped map.
	RegisterCustomJSONProvider("json", func() core.Ammo { return map[string]interface{}{} })

	register.Aggregator("phout", func(conf netsample.PhoutConfig) (core.Aggregator, error) {
		a, err := netsample.NewPhout(fs, conf)
		return netsample.WrapAggregator(a), err
	}, netsample.DefaultPhoutConfig)
	register.Aggregator("jsonlines", aggregator.NewJSONLinesAggregator, aggregator.DefaultJSONLinesAggregatorConfig)
	register.Aggregator("json", aggregator.NewJSONLinesAggregator, aggregator.DefaultJSONLinesAggregatorConfig) // TODO(skipor): should be done via alias, but we don't have them yet
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
	dataSinkConfigHooks   []PluginConfigStringHook
	dataSourceConfigHooks []PluginConfigStringHook
)

func AddSinkConfigHook(hook PluginConfigStringHook) {
	dataSinkConfigHooks = append(dataSinkConfigHooks, hook)
}

func AddSourceConfigHook(hook PluginConfigStringHook) {
	dataSourceConfigHooks = append(dataSourceConfigHooks, hook)
}

func RegisterCustomJSONProvider(name string, newAmmo func() core.Ammo) {
	register.Provider(name, func(conf provider.JSONProviderConfig) core.Provider {
		return provider.NewJSONProvider(newAmmo, conf)
	}, provider.DefaultJSONProviderConfig)
}

// sourceStringHook helps to decode string as core.DataSource plugin.
// Try use source hooks and use file as fallback.
func sourceStringHook(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if !isPluginOrFactory(dataSourceType, t) {
		return data, nil
	}
	if tag.Debug {
		zap.L().Debug("DataSource string hook triggered")
	}
	var (
		ok         bool
		pluginType string
		conf       map[string]interface{}
	)
	dataStr := data.(string)

	for _, hook := range dataSourceConfigHooks {
		ok, pluginType, conf = hook(dataStr)
		zap.L().Debug("Source hooked", zap.String("plugin", pluginType))
		if ok {
			break
		}
	}

	if !ok {
		zap.L().Debug("Consider source as a file", zap.String("source", dataStr))
		pluginType = fileDataKey
		conf = map[string]interface{}{
			"path": data,
		}
	}

	if conf == nil {
		conf = make(map[string]interface{})
	}
	conf[pluginconfig.PluginNameKey] = pluginType

	if tag.Debug {
		zap.L().Debug("Hooked DataSource config", zap.Any("config", conf))
	}
	return conf, nil
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
		zap.L().Debug("Sink hooked", zap.String("plugin", pluginType))
		if ok {
			break
		}
	}

	if !ok {
		zap.L().Debug("Consider sink as a file", zap.String("source", dataStr))
		pluginType = fileDataKey
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
