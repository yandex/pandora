// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package coretest

import (
	"github.com/onsi/gomega"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/lib/testutil"
)

func Decode(data string, result interface{}) {
	conf := testutil.ParseYAML(data)
	err := config.Decode(conf, result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}

func DecodeAndValidate(data string, result interface{}) {
	Decode(data, result)
	err := config.Validate(result)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
}
