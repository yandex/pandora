// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>
package phttp

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/onsi/ginkgo/extensions/table"
	"github.com/yandex/pandora/components/phttp/mocks"
	"github.com/yandex/pandora/core/aggregate/netsample"
)

var _ = Describe("Base", func() {
	var (
		base Base
		ammo *ammomock.Ammo
	)
	BeforeEach(func() {
		base = Base{}
		ammo = &ammomock.Ammo{}
	})

	Context("BindResultTo", func() {
		It("nil panics", func() {
			Expect(func() {
				base.Bind(nil)
			}).To(Panic())
		})
		It("second time panics", func() {
			res := &netsample.TestAggregator{}
			base.Bind(res)
			Expect(base.Aggregator).To(Equal(res))
			Expect(func() {
				base.Bind(&netsample.TestAggregator{})
			}).To(Panic())
		})
	})

	It("Shoot before bind panics", func() {
		base.Do = func(*http.Request) (_ *http.Response, _ error) {
			Fail("should not be called")
			return
		}
		am := &ammomock.Ammo{}
		am.On("Request").Return(nil, nil).Run(
			func(mock.Arguments) {
				Fail("should not be caled")
			})
		Expect(func() {
			base.Shoot(context.Background(), am)
		}).To(Panic())
	}, 1)

	Context("Shoot", func() {
		var (
			body io.ReadCloser

			ctx      context.Context
			am       *ammomock.Ammo
			req      *http.Request
			res      *http.Response
			sample   *netsample.Sample
			results  *netsample.TestAggregator
			shootErr error
		)
		BeforeEach(func() {
			ctx = context.Background()
			am = &ammomock.Ammo{}
			req = httptest.NewRequest("GET", "/1/2/3/4", nil)
			sample = netsample.Acquire("REQUEST")
			am.On("Request").Return(req, sample)
			results = &netsample.TestAggregator{}
			base.Bind(results)
		})

		JustBeforeEach(func() {
			res = &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       ioutil.NopCloser(body),
				Request:    req,
			}
			base.Shoot(ctx, am)
			Expect(results.Samples).To(HaveLen(1))
			shootErr = results.Samples[0].Err()
		})

		Context("Do ok", func() {
			BeforeEach(func() {
				body = ioutil.NopCloser(strings.NewReader("aaaaaaa"))
				base.Do = func(doReq *http.Request) (*http.Response, error) {
					Expect(doReq).To(Equal(req))
					return res, nil
				}
			})

			It("ammo sample sent to results", func() {
				Expect(results.Samples).To(HaveLen(1))
				Expect(results.Samples[0]).To(Equal(sample))
				Expect(sample.Tags()).To(Equal("REQUEST"))
				Expect(sample.ProtoCode()).To(Equal(res.StatusCode))
			})
			It("body read well", func() {
				Expect(shootErr).To(BeNil())
				_, err := body.Read([]byte{0})
				Expect(err).To(Equal(io.EOF), "body should be read fully")
			})
			Context("autotag options is set", func() {
				BeforeEach(func() {
					base.Config.AutoTag = 2
				})
				It("autotagged", func() {
					Expect(sample.Tags()).To(Equal("REQUEST|/1/2"))
				})
			})
			Context("Connect set", func() {
				var connectCalled, doCalled bool
				BeforeEach(func() {
					base.Connect = func(ctx context.Context) error {
						connectCalled = true
						return nil
					}
					oldDo := base.Do
					base.Do = func(r *http.Request) (*http.Response, error) {
						doCalled = true
						return oldDo(r)
					}
				})
				It("Connect called", func() {
					Expect(shootErr).To(BeNil())
					Expect(connectCalled).To(BeTrue())
					Expect(doCalled).To(BeTrue())
				})
			})
			Context("Connect failed", func() {
				connectErr := errors.New("connect error")
				BeforeEach(func() {
					base.Connect = func(ctx context.Context) error {
						// Connect should report fail in sample itself.
						s := netsample.Acquire("")
						s.SetErr(connectErr)
						results.Report(s)
						return connectErr
					}
				})
				It("Shoot failed", func() {
					Expect(shootErr).NotTo(BeNil())
					Expect(shootErr).To(Equal(connectErr))
				})
			})
		})
	})

	table.DescribeTable("autotag",
		func(path string, depth int, tag string) {
			URL := &url.URL{Path: path}
			Expect(autotag(depth, URL)).To(Equal(tag))
		},
		table.Entry("empty", "", 2, ""),
		table.Entry("root", "/", 2, "/"),
		table.Entry("exact depth", "/1/2", 2, "/1/2"),
		table.Entry("more depth", "/1/2", 3, "/1/2"),
		table.Entry("less depth", "/1/2", 1, "/1"),
	)

})
