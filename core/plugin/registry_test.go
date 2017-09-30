// Copyright (c) 2016 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"io"
	"reflect"

	"github.com/mitchellh/mapstructure"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("new default config container", func() {
	DescribeTable("expectation fail",
		func(constructor interface{}, newDefaultConfigOptional ...interface{}) {
			newDefaultConfig := getNewDefaultConfig(newDefaultConfigOptional)
			defer recoverExpectationFail()
			newDefaultConfigContainer(reflect.TypeOf(constructor), newDefaultConfig)
		},
		Entry("invalid type",
			func(int) ptestPlugin { return nil }),
		Entry("invalid ptr type",
			func(*int) ptestPlugin { return nil }),
		Entry("to many args",
			func(_, _ ptestConfig) ptestPlugin { return nil }),
		Entry("default without config",
			func() ptestPlugin { return nil }, func() *ptestConfig { return nil }),
		Entry("invalid default config",
			func(ptestConfig) ptestPlugin { return nil }, func() *ptestConfig { return nil }),
		Entry("default config accepts args",
			func(*ptestConfig) ptestPlugin { return nil }, func(int) *ptestConfig { return nil }),
	)

	DescribeTable("expectation ok",
		func(constructor interface{}, newDefaultConfigOptional ...interface{}) {
			newDefaultConfig := getNewDefaultConfig(newDefaultConfigOptional)
			container := newDefaultConfigContainer(reflect.TypeOf(constructor), newDefaultConfig)
			conf, err := container.Get(ptestFillConf)
			Expect(err).NotTo(HaveOccurred())
			Expect(conf).To(HaveLen(1))
			ptestExpectConfigValue(conf[0].Interface(), ptestFilledValue)
		},
		Entry("no default config",
			ptestNewConf),
		Entry("no default ptr config",
			ptestNewPtrConf),
		Entry("default config",
			ptestNewConf, ptestDefaultConf),
		Entry("default ptr config",
			ptestNewPtrConf, ptestNewDefaultPtrConf),
	)

	It("fill no config failed", func() {
		container := newDefaultConfigContainer(ptestNewErrType(), nil)
		_, err := container.Get(ptestFillConf)
		Expect(err).To(HaveOccurred())
	})
})

var _ = DescribeTable("register valid",
	func(
		newPluginImpl interface{},
		newDefaultConfigOptional ...interface{},
	) {
		Expect(func() {
			NewRegistry().ptestRegister(newPluginImpl, newDefaultConfigOptional...)
		}).NotTo(Panic())
	},
	Entry("return impl",
		func() *ptestImpl { return nil }),
	Entry("return interface",
		func() ptestPlugin { return nil }),
	Entry("super interface",
		func() interface {
			io.Writer
			ptestPlugin
		} {
			return nil
		}),
	Entry("struct config",
		func(ptestConfig) ptestPlugin { return nil }),
	Entry("struct ptr config",
		func(*ptestConfig) ptestPlugin { return nil }),
	Entry("default config",
		func(*ptestConfig) ptestPlugin { return nil },
		func() *ptestConfig { return nil }),
)

var _ = DescribeTable("register invalid",
	func(
		newPluginImpl interface{},
		newDefaultConfigOptional ...interface{},
	) {
		Expect(func() {
			defer recoverExpectationFail()
			NewRegistry().ptestRegister(newPluginImpl, newDefaultConfigOptional...)
		}).NotTo(Panic())
	},
	Entry("return not impl",
		func() ptestImpl { panic("") }),
	Entry("invalid config type",
		func(int) ptestPlugin { return nil }),
	Entry("invalid config ptr type",
		func(*int) ptestPlugin { return nil }),
	Entry("to many args",
		func(_, _ ptestConfig) ptestPlugin { return nil }),
	Entry("default without config",
		func() ptestPlugin { return nil }, func() *ptestConfig { return nil }),
	Entry("extra default config",
		func(*ptestConfig) ptestPlugin { return nil }, func() *ptestConfig { return nil }, 0),
	Entry("invalid default config",
		func(ptestConfig) ptestPlugin { return nil }, func() *ptestConfig { return nil }),
	Entry("default config accepts args",
		func(*ptestConfig) ptestPlugin { return nil }, func(int) *ptestConfig { return nil }),
)

var _ = Describe("registry", func() {
	It("register name collision panics", func() {
		r := NewRegistry()
		r.ptestRegister(ptestNewImpl)
		defer recoverExpectationFail()
		r.ptestRegister(ptestNewImpl)
	})
	It("lookup", func() {
		r := NewRegistry()
		r.ptestRegister(ptestNewImpl)
		Expect(r.Lookup(ptestType())).To(BeTrue())
		Expect(r.Lookup(reflect.TypeOf(0))).To(BeFalse())
		Expect(r.Lookup(reflect.TypeOf(&ptestImpl{}))).To(BeFalse())
		Expect(r.Lookup(reflect.TypeOf((*io.Writer)(nil)).Elem())).To(BeFalse())
	})

})

var _ = Describe("new", func() {
	type New func(r *Registry, fillConfOptional ...func(conf interface{}) error) (interface{}, error)
	var (
		r         *Registry
		testNew   New
		testNewOk = func(fillConfOptional ...func(conf interface{}) error) (pluginVal string) {
			plugin, err := testNew(r, fillConfOptional...)
			Expect(err).NotTo(HaveOccurred())
			return plugin.(*ptestImpl).Value
		}
	)
	BeforeEach(func() { r = NewRegistry() })
	runTestCases := func() {
		It("no conf", func() {
			r.ptestRegister(ptestNewImpl)
			Expect(testNewOk()).To(Equal(ptestInitValue))
		})
		It("nil error", func() {
			r.ptestRegister(func() (ptestPlugin, error) {
				return ptestNewImpl(), nil
			})
			Expect(testNewOk()).To(Equal(ptestInitValue))
		})
		It("non-nil error", func() {
			expectedErr := errors.New("fill conf err")
			r.ptestRegister(func() (ptestPlugin, error) {
				return nil, expectedErr
			})
			_, err := testNew(r)
			Expect(err).To(HaveOccurred())
			err = errors.Cause(err)
			Expect(expectedErr).To(Equal(err))
		})
		It("no conf, fill conf error", func() {
			r.ptestRegister(ptestNewImpl)
			expectedErr := errors.New("fill conf err")
			_, err := testNew(r, func(_ interface{}) error { return expectedErr })
			Expect(expectedErr).To(Equal(err))
		})
		It("no default", func() {
			r.ptestRegister(func(c ptestConfig) *ptestImpl { return &ptestImpl{c.Value} })
			Expect(testNewOk()).To(Equal(""))
		})
		It("default", func() {
			r.ptestRegister(ptestNewConf, ptestDefaultConf)
			Expect(testNewOk()).To(Equal(ptestDefaultValue))
		})
		It("fill conf default", func() {
			r.ptestRegister(ptestNewConf, ptestDefaultConf)
			Expect(testNewOk(ptestFillConf)).To(Equal(ptestFilledValue))
		})
		It("fill conf no default", func() {
			r.ptestRegister(ptestNewConf)
			Expect(testNewOk(ptestFillConf)).To(Equal(ptestFilledValue))
		})
		It("fill ptr conf no default", func() {
			r.ptestRegister(ptestNewPtrConf)
			Expect(testNewOk(ptestFillConf)).To(Equal(ptestFilledValue))
		})
		It("no default ptr conf not nil", func() {
			r.ptestRegister(ptestNewPtrConf)
			Expect("").To(Equal(testNewOk()))
		})
		It("nil default, conf not nil", func() {
			r.ptestRegister(ptestNewPtrConf, func() *ptestConfig { return nil })
			Expect(testNewOk(ptestFillConf)).To(Equal(ptestFilledValue))
		})
		It("fill nil default", func() {
			r.ptestRegister(ptestNewPtrConf, func() *ptestConfig { return nil })
			Expect(testNewOk(ptestFillConf)).To(Equal(ptestFilledValue))
		})
		It("more than one fill conf panics", func() {
			r.ptestRegister(ptestNewPtrConf)
			defer recoverExpectationFail()
			testNew(r, ptestFillConf, ptestFillConf)
		})
	}
	Context("use New", func() {
		BeforeEach(func() { testNew = (*Registry).ptestNew })
		runTestCases()

	})
	Context("use NewFactory", func() {
		BeforeEach(func() { testNew = (*Registry).ptestNewFactory })
		runTestCases()
	})

})

var _ = Describe("decode", func() {
	It("ok", func() {
		r := NewRegistry()
		const nameKey = "type"

		var hook mapstructure.DecodeHookFunc
		decode := func(input, result interface{}) error {
			decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
				DecodeHook:  hook,
				ErrorUnused: true,
				Result:      result,
			})
			if err != nil {
				return err
			}
			return decoder.Decode(input)
		}
		hook = mapstructure.ComposeDecodeHookFunc(
			func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
				if !r.Lookup(to) {
					return data, nil
				}
				// NOTE: could be map[interface{}]interface{} here.
				input := data.(map[string]interface{})
				// NOTE: should be case insensitive behaviour.
				pluginName := input[nameKey].(string)
				delete(input, nameKey)
				return r.New(to, pluginName, func(conf interface{}) error {
					// NOTE: should error, if conf has "type" field.
					return decode(input, conf)
				})
			})

		r.Register(ptestType(), "my-plugin", ptestNewConf, ptestDefaultConf)
		input := map[string]interface{}{
			"plugin": map[string]interface{}{
				nameKey: "my-plugin",
				"value": ptestFilledValue,
			},
		}
		type Config struct {
			Plugin ptestPlugin
		}
		var conf Config
		err := decode(input, &conf)
		Expect(err).NotTo(HaveOccurred())
		actualValue := conf.Plugin.(*ptestImpl).Value
		Expect(actualValue).To(Equal(ptestFilledValue))
	})

})

func recoverExpectationFail() {
	r := recover()
	Expect(r).NotTo(BeNil())
	Expect(r).To(ContainSubstring("expectation failed"))
}
