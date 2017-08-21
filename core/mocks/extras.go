// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coremock

import (
	"fmt"
	"unsafe"
)

// Implement Stringer, so when Aggregator is passed as arg to another mock call,
// it not read and data races not created.
func (_m *Aggregator) String() string {
	return fmt.Sprintf("coremock.Aggregator{%v}", unsafe.Pointer(_m))
}
