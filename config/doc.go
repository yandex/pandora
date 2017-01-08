// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MLP 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

// Package config provides advanced framework to decode and validate
// configuration structs.
// Package should not depend on other project packages.
// So hooks which have dependencies from other project package should be added
// from its packages via AddKindHook or AddTypeHook.
package config
