// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"testing"

	"github.com/yandex/pandora/lib/testutil"
)

func TestPhttp(t *testing.T) {
	testutil.RunSuite(t, "HTTP Suite")
}
