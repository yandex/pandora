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
			func(int) testPlugin { return nil }),
		Entry("invalid ptr type",
			func(*int) testPlugin { return nil }),
		Entry("to many args",
			func(_, _ testPluginConfig) testPlugin { return nil }),
		Entry("default without config",
			func() testPlugin { return nil }, func() *testPluginConfig { return nil }),
		Entry("invalid default config",
			func(testPluginConfig) testPlugin { return nil }, func() *testPluginConfig { return nil }),
		Entry("default config accepts args",
			func(*testPluginConfig) testPlugin { return nil }, func(int) *testPluginConfig { return nil }),
	)

	DescribeTable("expectation ok",
		func(constructor interface{}, newDefaultConfigOptional ...interface{}) {
			newDefaultConfig := getNewDefaultConfig(newDefaultConfigOptional)
			container := newDefaultConfigContainer(reflect.TypeOf(constructor), newDefaultConfig)
			conf, err := container.Get(fillTestPluginConf)
			Expect(err).NotTo(HaveOccurred())
			Expect(conf).To(HaveLen(1))
			expectConfigValue(conf[0].Interface(), testFilledValue)
		},
		Entry("no default config",
			newTestPluginImplConf),
		Entry("no default ptr config",
			newTestPluginImplPtrConf),
		Entry("default config",
			newTestPluginImplConf, newTestPluginDefaultConf),
		Entry("default ptr config",
			newTestPluginImplPtrConf, newTestPluginDefaultPtrConf),
	)

	It("fill no config failed", func() {
		container := newDefaultConfigContainer(testPluginFactoryType(), nil)
		_, err := container.Get(fillTestPluginConf)
		Expect(err).To(HaveOccurred())
	})
})

var _ = DescribeTable("register valid",
	func(
		newPluginImpl interface{},
		newDefaultConfigOptional ...interface{},
	) {
		Expect(func() {
			NewRegistry().testRegister(newPluginImpl, newDefaultConfigOptional...)
		}).NotTo(Panic())
	},
	Entry("return impl",
		func() *testPluginImpl { return nil }),
	Entry("return interface",
		func() testPlugin { return nil }),
	Entry("super interface",
		func() interface {
			io.Writer
			testPlugin
		} {
			return nil
		}),
	Entry("struct config",
		func(testPluginConfig) testPlugin { return nil }),
	Entry("struct ptr config",
		func(*testPluginConfig) testPlugin { return nil }),
	Entry("default config",
		func(*testPluginConfig) testPlugin { return nil },
		func() *testPluginConfig { return nil }),
)

var _ = DescribeTable("register invalid",
	func(
		newPluginImpl interface{},
		newDefaultConfigOptional ...interface{},
	) {
		Expect(func() {
			defer recoverExpectationFail()
			NewRegistry().testRegister(newPluginImpl, newDefaultConfigOptional...)
		}).NotTo(Panic())
	},
	Entry("return not impl",
		func() testPluginImpl { panic("") }),
	Entry("invalid config type",
		func(int) testPlugin { return nil }),
	Entry("invalid config ptr type",
		func(*int) testPlugin { return nil }),
	Entry("to many args",
		func(_, _ testPluginConfig) testPlugin { return nil }),
	Entry("default without config",
		func() testPlugin { return nil }, func() *testPluginConfig { return nil }),
	Entry("extra default config",
		func(*testPluginConfig) testPlugin { return nil }, func() *testPluginConfig { return nil }, 0),
	Entry("invalid default config",
		func(testPluginConfig) testPlugin { return nil }, func() *testPluginConfig { return nil }),
	Entry("default config accepts args",
		func(*testPluginConfig) testPlugin { return nil }, func(int) *testPluginConfig { return nil }),
)

var _ = Describe("registry", func() {
	It("register name collision panics", func() {
		r := NewRegistry()
		r.testRegister(newTestPluginImpl)
		defer recoverExpectationFail()
		r.testRegister(newTestPluginImpl)
	})
	It("lookup", func() {
		r := NewRegistry()
		r.testRegister(newTestPluginImpl)
		Expect(r.Lookup(testPluginType())).To(BeTrue())
		Expect(r.Lookup(reflect.TypeOf(0))).To(BeFalse())
		Expect(r.Lookup(reflect.TypeOf(&testPluginImpl{}))).To(BeFalse())
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
			return plugin.(*testPluginImpl).Value
		}
	)
	BeforeEach(func() { r = NewRegistry() })
	runTestCases := func() {
		It("no conf", func() {
			r.testRegister(newTestPluginImpl)
			Expect(testNewOk()).To(Equal(testInitValue))
		})
		It("nil error", func() {
			r.testRegister(func() (testPlugin, error) {
				return newTestPluginImpl(), nil
			})
			Expect(testNewOk()).To(Equal(testInitValue))
		})
		It("non-nil error", func() {
			expectedErr := errors.New("fill conf err")
			r.testRegister(func() (testPlugin, error) {
				return nil, expectedErr
			})
			_, err := testNew(r)
			Expect(err).To(HaveOccurred())
			err = errors.Cause(err)
			Expect(expectedErr).To(Equal(err))
		})
		It("no conf, fill conf error", func() {
			r.testRegister(newTestPluginImpl)
			expectedErr := errors.New("fill conf err")
			_, err := testNew(r, func(_ interface{}) error { return expectedErr })
			Expect(expectedErr).To(Equal(err))
		})
		It("no default", func() {
			r.testRegister(func(c testPluginConfig) *testPluginImpl { return &testPluginImpl{c.Value} })
			Expect(testNewOk()).To(Equal(""))
		})
		It("default", func() {
			r.testRegister(newTestPluginImplConf, newTestPluginDefaultConf)
			Expect(testNewOk()).To(Equal(testDefaultValue))
		})
		It("fill conf default", func() {
			r.testRegister(newTestPluginImplConf, newTestPluginDefaultConf)
			Expect("conf").To(Equal(testNewOk(fillTestPluginConf)))
		})
		It("fill conf no default", func() {
			r.testRegister(newTestPluginImplConf)
			Expect("conf").To(Equal(testNewOk(fillTestPluginConf)))
		})
		It("fill ptr conf no default", func() {
			r.testRegister(newTestPluginImplPtrConf)
			Expect("conf").To(Equal(testNewOk(fillTestPluginConf)))
		})
		It("no default ptr conf not nil", func() {
			r.testRegister(newTestPluginImplPtrConf)
			Expect("").To(Equal(testNewOk()))
		})
		It("nil default, conf not nil", func() {
			r.testRegister(newTestPluginImplPtrConf, func() *testPluginConfig { return nil })
			Expect("").To(Equal(testNewOk()))
		})
		It("fill nil default", func() {
			r.testRegister(newTestPluginImplPtrConf, func() *testPluginConfig { return nil })
			Expect("conf").To(Equal(testNewOk(fillTestPluginConf)))
		})
		It("more than one fill conf panics", func() {
			r.testRegister(newTestPluginImplPtrConf)
			defer recoverExpectationFail()
			testNew(r, fillTestPluginConf, fillTestPluginConf)
		})
	}
	Context("use New", func() {
		BeforeEach(func() { testNew = (*Registry).testNew })
		runTestCases()

	})
	Context("use NewFactory", func() {
		BeforeEach(func() { testNew = (*Registry).testNewFactory })
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

		r.Register(testPluginType(), "my-plugin", newTestPluginImplConf, newTestPluginDefaultConf)
		input := map[string]interface{}{
			"plugin": map[string]interface{}{
				nameKey: "my-plugin",
				"value": testFilledValue,
			},
		}
		type Config struct {
			Plugin testPlugin
		}
		var conf Config
		err := decode(input, &conf)
		Expect(err).NotTo(HaveOccurred())
		actualValue := conf.Plugin.(*testPluginImpl).Value
		Expect(actualValue).To(Equal(testFilledValue))
	})

})

func recoverExpectationFail() {
	r := recover()
	Expect(r).NotTo(BeNil())
	Expect(r).To(ContainSubstring("expectation failed"))
}
