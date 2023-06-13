// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package http

import (
	"net/http"

	phttp "github.com/yandex/pandora/components/guns/http"
	"github.com/yandex/pandora/components/providers/base"
)

type Request interface {
	http.Request
}

var _ phttp.Ammo = (*base.Ammo)(nil)
