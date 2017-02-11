// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/yandex/pandora/config"
)

var _ = Describe("Base", func() {
	It("GunClientConfig decode", func() {
		conf := NewDefaultHTTPGunClientConfig()
		data := map[interface{}]interface{}{
			"target": "test-trbo01e.haze.yandex.net:3000",
		}
		err := config.DecodeAndValidate(data, &conf)
		Expect(err).To(BeNil())
	})
})
