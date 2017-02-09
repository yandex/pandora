package phttp

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/yandex/pandora/aggregate"
	"github.com/yandex/pandora/ammo/mocks"
)

var _ = Describe("Base", func() {
	var (
		base Base
		ammo *ammomocks.HTTP
	)
	BeforeEach(func() {
		base = Base{}
		ammo = &ammomocks.HTTP{}
	})

	Context("BindResultTo", func() {
		It("nil panics", func() {
			Expect(func() {
				base.BindResultsTo(nil)
			}).To(Panic())
		})
		It("second time panics", func() {
			res := make(chan<- *aggregate.Sample)
			base.BindResultsTo(res)
			Expect(base.Results).To(Equal(res))
			Expect(func() {
				base.BindResultsTo(make(chan<- *aggregate.Sample))
			}).To(Panic())
		})
	})

	It("Shoot before bind panics", func() {
		base.Do = func(*http.Request) (_ *http.Response, _ error) {
			Fail("should not be called")
			return
		}
		am := &ammomocks.HTTP{}
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
			am       *ammomocks.HTTP
			req      *http.Request
			res      *http.Response
			sample   *aggregate.Sample
			results  chan *aggregate.Sample
			shootErr error
		)
		BeforeEach(func() {
			ctx = context.Background()
			am = &ammomocks.HTTP{}
			req = httptest.NewRequest("GET", "/", nil)
			sample = aggregate.AcquireSample("REQUEST")
			am.On("Request").Return(req, sample)
			results = make(chan *aggregate.Sample, 1) // Results buffered.
			base.BindResultsTo(results)
		})

		JustBeforeEach(func() {
			res = &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       ioutil.NopCloser(body),
				Request:    req,
			}
			shootErr = base.Shoot(ctx, am)
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
				var gotSample *aggregate.Sample
				Eventually(results).Should(Receive(&gotSample))
				Expect(gotSample).To(Equal(sample))
				Expect(sample.ProtoCode()).To(Equal(res.StatusCode))
			})
			It("body read well", func() {
				Expect(shootErr).To(BeNil())
				_, err := body.Read([]byte{0})
				Expect(err).To(Equal(io.EOF), "body should be read fully")
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
})
