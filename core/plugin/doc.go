// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

// Package plugin provides a generic inversion of control model for making
// extensible Go packages, libraries, and applications. Like
// github.com/progrium/go-extpoints, but reflect based: doesn't require code
// generation, but have more overhead; provide more flexibility, but less type
// safety. It allows to register constructor for some plugin interface, and create
// new plugin instances or plugin instance factories.
// Main feature is flexible plugin configuration: plugin factory can
// accept config struct, that could be filled by passed hook. Config default
// values could be provided by registering default config factory.
// Such flexibility can be used to decode structured text (json/yaml/etc) into
// struct.
//
// Type expectations.
// Here and bellow we mean by <someTypeName> some type expectations.
// [some type signature part] means that this part of type signature is optional.
//
// Plugin type, let's label it as <plugin>, should be interface.
// Registered plugin constructor should be one of: <newPlugin> or <newFactory>.
// <newPlugin> should have type func([config <configType>]) (<pluginImpl>[, error]).
// <newFactory> should have type func([config <configType]) (func() (<pluginImpl>[, error])[, error]).
// <pluginImpl> should be assignable to <plugin>.
// That is, <plugin> methods should be subset <pluginImpl> methods. In other words, <pluginImpl> should be
// some <plugin> implementation, <plugin> or interface, that contains <plugin> methods as subset.
// <configType> type should be struct or struct pointer.
package plugin
