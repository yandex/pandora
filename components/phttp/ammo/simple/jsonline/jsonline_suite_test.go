// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package jsonline

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/spf13/afero"

	"github.com/yandex/pandora/components/phttp/ammo/simple"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/ginkgoutil"
)

func TestJsonline(t *testing.T) {
	ginkgoutil.RunSuite(t, "Jsonline Suite")
}

const testFile = "./ammo.jsonline"

// testData holds jsonline.data that contains in testFile
var testData = []data{
	{
		Host:    "example.com",
		Method:  "GET",
		Uri:     "/00",
		Headers: map[string]string{"Accept": "*/*", "Accept-Encoding": "gzip, deflate", "User-Agent": "Pandora/0.0.1"},
	},
	{
		Host:    "ya.ru",
		Method:  "HEAD",
		Uri:     "/01",
		Headers: map[string]string{"Accept": "*/*", "Accept-Encoding": "gzip, brotli", "User-Agent": "YaBro/0.1"},
		Tag:     "head",
	},
}

var testFs = newTestFs()

func newTestFs() afero.Fs {
	fs := afero.NewMemMapFs()
	file, err := fs.Create(testFile)
	if err != nil {
		panic(err)
	}
	encoder := json.NewEncoder(file)
	for _, d := range testData {
		err := encoder.Encode(d)
		if err != nil {
			panic(err)
		}
	}
	return afero.NewReadOnlyFs(fs)
}

func newTestProvider(conf Config) *Provider {
	return NewProvider(testFs, conf)
}

var _ = Describe("data", func() {
	It("decoded well", func() {
		data := data{
			Host:    "ya.ru",
			Method:  "GET",
			Uri:     "/00",
			Headers: map[string]string{"A": "a", "B": "b"},
			Tag:     "tag",
		}
		req, err := data.ToRequest()
		Expect(err).To(BeNil())
		Expect(*req).To(MatchFields(IgnoreExtras, Fields{
			"Proto":      Equal("HTTP/1.1"),
			"ProtoMajor": Equal(1),
			"ProtoMinor": Equal(1),
			"URL": PointTo(MatchFields(IgnoreExtras, Fields{
				"Scheme": Equal("http"),
				"Host":   Equal(data.Host),
				"Path":   Equal(data.Uri),
			})),
			"Header": Equal(http.Header{
				"A": []string{"a"},
				"B": []string{"b"},
			}),
			"Method": Equal(data.Method),
		}))
	})
})

var _ = Describe("provider start", func() {
	It("ok", func() {
		p := newTestProvider(Config{File: testFile})
		ctx, cancel := context.WithCancel(context.Background())
		errch := make(chan error)
		go func() { errch <- p.Run(ctx, core.ProviderDeps{}) }()
		Expect(errch).NotTo(Receive())
		cancel()
		var err error
		Eventually(errch).Should(Receive(&err))
		Expect(err).NotTo(HaveOccurred())
	})

	It("fail", func() {
		p := newTestProvider(Config{File: "no_such_file"})
		err := p.Run(context.Background(), core.ProviderDeps{})
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("provider decode", func() {
	var (
		// Configured in BeforeEach.
		conf             Config // File always is testFile.
		expectedStartErr error
		successReceives  int // How much ammo should be received.

		provider *Provider
		cancel   context.CancelFunc
		errch    chan error
		ammos    []*simple.Ammo
	)

	BeforeEach(func() {
		conf = Config{}
		expectedStartErr = nil
		ammos = nil
		successReceives = 0
	})

	JustBeforeEach(func() {
		conf.File = testFile
		provider = newTestProvider(conf)
		errch = make(chan error)
		var ctx context.Context
		ctx, cancel = context.WithCancel(context.Background())
		go func() { errch <- provider.Run(ctx, core.ProviderDeps{}) }()
		Expect(errch).NotTo(Receive())

		for i := 0; i < successReceives; i++ {
			am, ok := provider.Acquire()
			Expect(ok).To(BeTrue())
			ammos = append(ammos, am.(*simple.Ammo))
		}
	}, 1)

	AfterEach(func() {
		_, ok := provider.Acquire()
		Expect(ok).To(BeFalse(), "All ammo should be readed.")
		defer cancel()
		var err error
		Eventually(errch).Should(Receive(&err))
		if expectedStartErr == nil {
			Expect(err).To(BeNil())
		} else {
			Expect(err).To(Equal(expectedStartErr))
		}
		for i := 0; i < len(ammos); i++ {
			expectedData := testData[i%len(testData)]
			expectedReq, err := expectedData.ToRequest()
			Expect(err).To(BeNil())
			ammo := ammos[i]
			req, ss := ammo.Request()
			Expect(*req).To(MatchFields(IgnoreExtras, Fields{
				"Proto":      Equal(expectedReq.Proto),
				"ProtoMajor": Equal(expectedReq.ProtoMajor),
				"ProtoMinor": Equal(expectedReq.ProtoMinor),
				"URL": PointTo(MatchFields(IgnoreExtras, Fields{
					"Scheme": Equal(expectedReq.URL.Scheme),
					"Host":   Equal(expectedReq.URL.Host),
					"Path":   Equal(expectedReq.URL.Path),
				})),
				"Header": Equal(expectedReq.Header),
				"Method": Equal(expectedReq.Method),
				"Body":   Equal(expectedReq.Body),
			}))
			Expect(ss.Tags()).To(Equal(expectedData.Tag))
		}
	})

	Context("unlimited", func() {
		BeforeEach(func() {
			successReceives = 5 * len(testData)
		})
		It("ok", func() {
			cancel()
			expectedStartErr = nil
			Eventually(provider.Sink, time.Second, time.Millisecond).Should(BeClosed())
		})
	})

	Context("limit set", func() {
		BeforeEach(func() {
			conf.Passes = 2
			successReceives = len(testData) * conf.Passes
		})
		It("ok", func() {
		})
	})

	Context("passes set", func() {
		BeforeEach(func() {
			conf.Passes = 10
			conf.Limit = 5
			successReceives = conf.Limit
		})
		It("ok", func() {})
	})

})

func Benchmark(b *testing.B) {
	RegisterTestingT(b)
	jsonDoc, err := json.Marshal(testData[0])
	Expect(err).To(BeNil())
	pool := sync.Pool{
		New: func() interface{} { return &simple.Ammo{} },
	}
	b.Run("Decode", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			decodeAmmo(jsonDoc, &simple.Ammo{})
		}
	})
	b.Run("DecodeWithPool", func(b *testing.B) {
		for n := 0; n < b.N; n++ {
			h := pool.Get().(*simple.Ammo)
			decodeAmmo(jsonDoc, h)
			pool.Put(h)
		}
	})
}
