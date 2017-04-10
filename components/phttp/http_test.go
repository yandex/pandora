// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/yandex/pandora/components/phttp/mocks"
	"github.com/yandex/pandora/core/aggregate/netsample"
	"github.com/yandex/pandora/core/config"
)

var _ = Describe("Base", func() {
	It("GunClientConfig decode", func() {
		conf := NewDefaultHTTPGunConfig()
		data := map[interface{}]interface{}{
			"target": "test-trbo01e.haze.yandex.net:3000",
		}
		err := config.DecodeAndValidate(data, &conf)
		Expect(err).To(BeNil())
	})

	It("integration", func() {
		const host = "example.com"
		const path = "/smth"
		expectedReq, err := http.NewRequest("GET", "http://"+host+path, nil)
		expectedReq.Host = "" // Important. Ammo may have empty host.
		Expect(err).To(BeNil())
		var actualReq *http.Request
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			rw.WriteHeader(http.StatusOK)
			actualReq = req
		}))
		defer server.Close()
		conf := NewDefaultHTTPGunConfig()
		conf.Gun.Target = strings.TrimPrefix(server.URL, "http://")
		results := &netsample.TestAggregator{}
		httpGun := NewHTTPGun(conf)
		httpGun.Bind(results)

		am := newTestAmmo(expectedReq)
		err = httpGun.Shoot(context.Background(), am)
		Expect(err).To(BeNil())

		Expect(*actualReq).To(MatchFields(IgnoreExtras, Fields{
			"Method": Equal("GET"),
			"Proto":  Equal("HTTP/1.1"),
			"Host":   Equal(host), // Not server host, but host from ammo.
			"URL": PointTo(MatchFields(IgnoreExtras, Fields{
				"Host": BeEmpty(), // Server request.
				"Path": Equal(path),
			})),
		}))
	})
})

func newTestAmmo(req *http.Request) Ammo {
	ammo := &ammomock.Ammo{}
	ammo.On("Request").Return(req, netsample.Acquire("REQUEST"))
	return ammo
}
