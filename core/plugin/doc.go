// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

// Package plugin provides a generic inversion of control model for making
// extensible Go packages, libraries, and applications. Like
// github.com/progrium/go-extpoints, but reflect based: doesn't require code
// generation, but have more overhead; provide more flexibility, but less type
// safety. It allows to register factory for some plugin interface, and create
// new plugin instances by registered factory.
// Main feature is flexible plugin configuration: plugin factory can
// accept config struct, that could be filled by passed hook. Config default
// values could be provided by registering default config factory.
// Such flexibility can be used to decode structured text (json/yaml/etc) into
// struct with plugin interface fields.
//
// Type expectations.
// Plugin factory type should be:
// func([config <configType>]) (<pluginImpl>[, error])
// where configType kind is struct or struct pointer, and pluginImpl implements
// plugin interface. Plugin factory will never receive nil config, even there
// are no registered default config factory, or default config is nil. Config
// will be pointer to zero config in such case.
// If plugin factory receive config argument, default config factory can be
// registered. Default config factory type should be: is func() <configType>.
// Default config factory is optional. If no default config factory has been
// registered, than plugin factory will receive zero config (zero struct or
// pointer to zero struct).
//
// Note, that plugin interface type could be taken as reflect.TypeOf((*PluginInterface)(nil)).Elem().
// FIXME: doc plugin factory
package plugin
