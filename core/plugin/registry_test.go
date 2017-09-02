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

var _ = DescribeTable("register valid",
	func(
		newPluginImpl interface{},
		newDefaultConfigOptional ...interface{},
	) {
		Expect(func() {
			newTypeRegistry().testRegister(newPluginImpl, newDefaultConfigOptional...)
		}).NotTo(Panic())
	},
	Entry("return impl",
		func() *testPluginImpl { return nil }),
	Entry("return interface",
		func() testPluginInterface { return nil }),
	Entry("super interface",
		func() interface {
			io.Writer
			testPluginInterface
		} {
			return nil
		}),
	Entry("struct config",
		func(testPluginImplConfig) testPluginInterface { return nil }),
	Entry("struct ptr config",
		func(*testPluginImplConfig) testPluginInterface { return nil }),
	Entry("default config",
		func(*testPluginImplConfig) testPluginInterface { return nil },
		func() *testPluginImplConfig { return nil }),
)

var _ = DescribeTable("register invalid",
	func(
		newPluginImpl interface{},
		newDefaultConfigOptional ...interface{},
	) {
		Expect(func() {
			defer recoverExpectationFail()
			newTypeRegistry().testRegister(newPluginImpl, newDefaultConfigOptional...)
		}).NotTo(Panic())
	},
	Entry("return not impl",
		func() testPluginImpl { panic("") }),
	Entry("invalid config type",
		func(int) testPluginInterface { return nil }),
	Entry("invalid config ptr type",
		func(*int) testPluginInterface { return nil }),
	Entry("to many args",
		func(_, _ testPluginImplConfig) testPluginInterface { return nil }),
	Entry("default without config",
		func() testPluginInterface { return nil }, func() *testPluginImplConfig { return nil }),
	Entry("extra deafult config",
		func(*testPluginImplConfig) testPluginInterface { return nil }, func() *testPluginImplConfig { return nil }, 0),
	Entry("invalid default config",
		func(testPluginImplConfig) testPluginInterface { return nil }, func() *testPluginImplConfig { return nil }),
	Entry("default config accepts args",
		func(*testPluginImplConfig) testPluginInterface { return nil }, func(int) *testPluginImplConfig { return nil }),
)

var _ = Describe("registry", func() {
	It("register name collision panics", func() {
		r := newTypeRegistry()
		r.testRegister(newTestPlugin)
		defer recoverExpectationFail()
		r.testRegister(newTestPlugin)
	})
	It("lookup", func() {
		r := newTypeRegistry()
		r.testRegister(newTestPlugin)
		Expect(r.Lookup(testPluginType())).To(BeTrue())
		Expect(r.Lookup(reflect.TypeOf(0))).To(BeFalse())
		Expect(r.Lookup(reflect.TypeOf(&testPluginImpl{}))).To(BeFalse())
		Expect(r.Lookup(reflect.TypeOf((*io.Writer)(nil)).Elem())).To(BeFalse())
	})

})

var _ = Describe("new", func() {
	type New func(r typeRegistry, fillConfOptional ...func(conf interface{}) error) (interface{}, error)
	var (
		r         typeRegistry
		testNew   New
		testNewOk = func(fillConfOptional ...func(conf interface{}) error) (pluginVal string) {
			plugin, err := testNew(r, fillConfOptional...)
			Expect(err).NotTo(HaveOccurred())
			return plugin.(*testPluginImpl).Value
		}
	)
	BeforeEach(func() { r = newTypeRegistry() })
	runTestCases := func() {
		It("no conf", func() {
			r.testRegister(newTestPlugin)
			Expect(testNewOk()).To(Equal(testInitValue))
		})
		It("nil error", func() {
			r.testRegister(func() (testPluginInterface, error) {
				return newTestPlugin(), nil
			})
			Expect(testNewOk()).To(Equal(testInitValue))
		})
		It("non-nil error", func() {
			expectedErr := errors.New("fill conf err")
			r.testRegister(func() (testPluginInterface, error) {
				return nil, expectedErr
			})
			_, err := testNew(r)
			Expect(err).To(HaveOccurred())
			err = errors.Cause(err)
			Expect(expectedErr).To(Equal(err))
		})
		It("no conf, fill conf error", func() {
			r.testRegister(newTestPlugin)
			expectedErr := errors.New("fill conf err")
			_, err := testNew(r, func(_ interface{}) error { return expectedErr })
			Expect(expectedErr).To(Equal(err))
		})
		It("no default", func() {
			r.testRegister(func(c testPluginImplConfig) *testPluginImpl { return &testPluginImpl{c.Value} })
			Expect(testNewOk()).To(Equal(""))
		})
		It("default", func() {
			r.testRegister(newTestPluginConf, newTestPluginDefaultConf)
			Expect(testNewOk()).To(Equal(testDefaultValue))
		})
		It("fill conf default", func() {
			r.testRegister(newTestPluginConf, newTestPluginDefaultConf)
			Expect("conf").To(Equal(testNewOk(fillTestPluginConf)))
		})
		It("fill conf no default", func() {
			r.testRegister(newTestPluginConf)
			Expect("conf").To(Equal(testNewOk(fillTestPluginConf)))
		})
		It("fill ptr conf no default", func() {
			r.testRegister(newTestPluginPtrConf)
			Expect("conf").To(Equal(testNewOk(fillTestPluginConf)))
		})
		It("no default ptr conf not nil", func() {
			r.testRegister(newTestPluginPtrConf)
			Expect("").To(Equal(testNewOk()))
		})
		It("nil default, conf not nil", func() {
			r.testRegister(newTestPluginPtrConf, func() *testPluginImplConfig { return nil })
			Expect("").To(Equal(testNewOk()))
		})
		It("fill nil default", func() {
			r.testRegister(newTestPluginPtrConf, func() *testPluginImplConfig { return nil })
			Expect("conf").To(Equal(testNewOk(fillTestPluginConf)))
		})
		It("more than one fill conf panics", func() {
			r.testRegister(newTestPluginPtrConf)
			defer recoverExpectationFail()
			testNew(r, fillTestPluginConf, fillTestPluginConf)
		})
	}
	Context("use New", func() {
		BeforeEach(func() { testNew = typeRegistry.testNew })
		runTestCases()

	})
	Context("use NewFactory", func() {
		BeforeEach(func() { testNew = typeRegistry.testNewFactory })
		runTestCases()
	})

})

var _ = Describe("decode", func() {
	It("ok", func() {
		r := newTypeRegistry()
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

		r.Register(testPluginType(), "my-plugin", newTestPluginConf, newTestPluginDefaultConf)
		input := map[string]interface{}{
			"plugin": map[string]interface{}{
				nameKey: "my-plugin",
				"value": testConfValue,
			},
		}
		type Config struct {
			Plugin testPluginInterface
		}
		var conf Config
		err := decode(input, &conf)
		Expect(err).NotTo(HaveOccurred())
		actualValue := conf.Plugin.(*testPluginImpl).Value
		Expect(actualValue).To(Equal(testConfValue))
	})

})

func recoverExpectationFail() {
	r := recover()
	Expect(r).NotTo(BeNil())
	Expect(r).To(ContainSubstring("expectation failed"))
}
