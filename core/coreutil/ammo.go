// Copyright (c) 2018 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coreutil

import (
	"reflect"

	"github.com/yandex/pandora/core"
)

// ResetReusedAmmo sets to zero any ammo.
// Used by core.Provider implementations that accepts generic type, and need to clean reused ammo
// before fill with fresh data.
func ResetReusedAmmo(ammo core.Ammo) {
	if resettable, ok := ammo.(core.ResettableAmmo); ok {
		resettable.Reset()
		return
	}
	elem := reflect.ValueOf(ammo).Elem()
	elem.Set(reflect.Zero(elem.Type()))
}
