// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package phttp

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"golang.org/x/net/http2"

	ammomock "github.com/yandex/pandora/components/phttp/mocks"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/config"
)

var _ = Describe("BaseGun", func() {
	It("GunClientConfig decode", func() {
		conf := DefaultHTTPGunConfig()
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
		conf := DefaultHTTPGunConfig()
		conf.Gun.Target = strings.TrimPrefix(server.URL, "http://")
		results := &netsample.TestAggregator{}
		httpGun := NewHTTPGun(conf)
		httpGun.Bind(results, testDeps())

		am := newAmmoReq(expectedReq)
		httpGun.Shoot(am)
		Expect(results.Samples[0].Err()).To(BeNil())

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

func newAmmoURL(url string) Ammo {
	req, err := http.NewRequest("GET", url, nil)
	Expect(err).NotTo(HaveOccurred())
	return newAmmoReq(req)
}

func newAmmoReq(req *http.Request) Ammo {
	ammo := &ammomock.Ammo{}
	ammo.On("Request").Return(req, netsample.Acquire("REQUEST"))
	return ammo
}

var _ = Describe("HTTP", func() {
	itOk := func(https bool) {
		var isServed atomic.Bool
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			Expect(req.Header.Get("Accept-Encoding")).To(BeEmpty())
			rw.WriteHeader(http.StatusOK)
			isServed.Store(true)
		}))
		if https {
			server.StartTLS()
		} else {
			server.Start()
		}
		defer server.Close()
		conf := DefaultHTTPGunConfig()
		conf.Gun.Target = server.Listener.Addr().String()
		conf.Gun.SSL = https
		gun := NewHTTPGun(conf)
		var aggr netsample.TestAggregator
		gun.Bind(&aggr, testDeps())
		gun.Shoot(newAmmoURL("/"))

		Expect(aggr.Samples).To(HaveLen(1))
		Expect(aggr.Samples[0].ProtoCode()).To(Equal(http.StatusOK))
		Expect(isServed.Load()).To(BeTrue())
	}
	It("http ok", func() { itOk(false) })
	It("https ok", func() { itOk(true) })

	itRedirect := func(redirect bool) {
		server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/redirect" {
				rw.Header().Add("Location", "/")
				rw.WriteHeader(http.StatusMovedPermanently)
			} else {
				rw.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()
		conf := DefaultHTTPGunConfig()
		conf.Gun.Target = server.Listener.Addr().String()
		conf.Client.Redirect = redirect
		gun := NewHTTPGun(conf)
		var aggr netsample.TestAggregator
		gun.Bind(&aggr, testDeps())
		gun.Shoot(newAmmoURL("/redirect"))

		Expect(aggr.Samples).To(HaveLen(1))
		expectedCode := http.StatusMovedPermanently
		if redirect {
			expectedCode = http.StatusOK
		}
		Expect(aggr.Samples[0].ProtoCode()).To(Equal(expectedCode))
	}
	It("not follow redirects by default", func() { itRedirect(false) })
	It("follow redirects if option set ", func() { itRedirect(true) })

	It("not support HTTP2", func() {
		server := newHTTP2TestServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if isHTTP2Request(req) {
				rw.WriteHeader(http.StatusForbidden)
			} else {
				rw.WriteHeader(http.StatusOK)
			}
		}))
		defer server.Close()

		// Test, that configured server serves HTTP2 well.
		http2OnlyClient := http.Client{
			Transport: &http2.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}}
		res, err := http2OnlyClient.Get(server.URL)
		Expect(err).NotTo(HaveOccurred())
		Expect(res.StatusCode).To(Equal(http.StatusForbidden))

		conf := DefaultHTTPGunConfig()
		conf.Gun.Target = server.Listener.Addr().String()
		conf.Gun.SSL = true
		gun := NewHTTPGun(conf)
		var results netsample.TestAggregator
		gun.Bind(&results, testDeps())
		gun.Shoot(newAmmoURL("/"))

		Expect(results.Samples).To(HaveLen(1))
		Expect(results.Samples[0].ProtoCode()).To(Equal(http.StatusOK))
	})

})

var _ = Describe("HTTP/2", func() {
	It("HTTP/2 ok", func() {
		server := newHTTP2TestServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			if isHTTP2Request(req) {
				rw.WriteHeader(http.StatusOK)
			} else {
				rw.WriteHeader(http.StatusForbidden)
			}
		}))
		defer server.Close()
		conf := DefaultHTTP2GunConfig()
		conf.Gun.Target = server.Listener.Addr().String()
		gun, _ := NewHTTP2Gun(conf)
		var results netsample.TestAggregator
		gun.Bind(&results, testDeps())
		gun.Shoot(newAmmoURL("/"))
		Expect(results.Samples[0].ProtoCode()).To(Equal(http.StatusOK))
	})

	It("HTTP/1.1 panic", func() {
		server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			zap.S().Info("Served")
		}))
		defer server.Close()
		conf := DefaultHTTP2GunConfig()
		conf.Gun.Target = server.Listener.Addr().String()
		gun, _ := NewHTTP2Gun(conf)
		var results netsample.TestAggregator
		gun.Bind(&results, testDeps())
		var r interface{}
		func() {
			defer func() {
				r = recover()
			}()
			gun.Shoot(newAmmoURL("/"))
		}()
		Expect(r).NotTo(BeNil())
		Expect(r).To(ContainSubstring(notHTTP2PanicMsg))
	})

	It("no SSL construction fails", func() {
		server := httptest.NewTLSServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			zap.S().Info("Served")
		}))
		defer server.Close()
		conf := DefaultHTTP2GunConfig()
		conf.Gun.Target = server.Listener.Addr().String()
		conf.Gun.SSL = false
		_, err := NewHTTP2Gun(conf)
		Expect(err).To(HaveOccurred())
	})

})

func isHTTP2Request(req *http.Request) bool {
	return checkHTTP2(req.TLS) == nil
}

func newHTTP2TestServer(handler http.Handler) *httptest.Server {
	server := httptest.NewUnstartedServer(handler)
	http2.ConfigureServer(server.Config, nil)
	server.TLS = server.Config.TLSConfig // StartTLS takes TLS configuration from that field.
	server.StartTLS()
	return server
}
