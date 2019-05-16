// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"errors"
	"fmt"
	"reflect"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("plugin constructor", func() {
	DescribeTable("expectations failed",
		func(newPlugin interface{}) {
			defer recoverExpectationFail()
			newPluginConstructor(ptestType(), newPlugin)
		},
		Entry("not func",
			errors.New("that is not constructor")),
		Entry("not implements",
			func() struct{} { panic("") }),
		Entry("too many args",
			func(_, _ ptestConfig) ptestPlugin { panic("") }),
		Entry("too many return valued",
			func() (_ ptestPlugin, _, _ error) { panic("") }),
		Entry("second return value is not error",
			func() (_, _ ptestPlugin) { panic("") }),
	)

	Context("new plugin", func() {
		newPlugin := func(newPlugin interface{}, maybeConf []reflect.Value) (interface{}, error) {
			testee := newPluginConstructor(ptestType(), newPlugin)
			return testee.NewPlugin(maybeConf)
		}

		It("", func() {
			plugin, err := newPlugin(ptestNew, nil)
			Expect(err).NotTo(HaveOccurred())
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("more that plugin", func() {
			plugin, err := newPlugin(ptestNewMoreThan, nil)
			Expect(err).NotTo(HaveOccurred())
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("config", func() {
			plugin, err := newPlugin(ptestNewConf, confToMaybe(ptestDefaultConf()))
			Expect(err).NotTo(HaveOccurred())
			ptestExpectConfigValue(plugin, ptestDefaultValue)
		})

		It("failed", func() {
			plugin, err := newPlugin(ptestNewErrFailing, nil)
			Expect(err).To(Equal(ptestCreateFailedErr))
			Expect(plugin).To(BeNil())
		})
	})

	Context("new factory", func() {
		newFactoryOK := func(newPlugin interface{}, factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) interface{} {
			testee := newPluginConstructor(ptestType(), newPlugin)
			factory, err := testee.NewFactory(factoryType, getMaybeConf)
			Expect(err).NotTo(HaveOccurred())
			return factory
		}

		It("same type - no wrap", func() {
			factory := newFactoryOK(ptestNew, ptestNewType(), nil)
			expectSameFunc(factory, ptestNew)
		})

		It(" new impl", func() {
			factory := newFactoryOK(ptestNewImpl, ptestNewType(), nil)
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			plugin := f()
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("more than", func() {
			factory := newFactoryOK(ptestNewMoreThan, ptestNewType(), nil)
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			plugin := f()
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("add err", func() {
			factory := newFactoryOK(ptestNew, ptestNewErrType(), nil)
			f, ok := factory.(func() (ptestPlugin, error))
			Expect(ok).To(BeTrue())
			plugin, err := f()
			Expect(err).NotTo(HaveOccurred())
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("trim nil err", func() {
			factory := newFactoryOK(ptestNewErr, ptestNewType(), nil)
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			plugin := f()
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("config", func() {
			factory := newFactoryOK(ptestNewConf, ptestNewType(), confToGetMaybe(ptestDefaultConf()))
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			plugin := f()
			ptestExpectConfigValue(plugin, ptestDefaultValue)
		})

		It("new factory, get config failed", func() {
			factory := newFactoryOK(ptestNewConf, ptestNewErrType(), errToGetMaybe(ptestConfigurationFailedErr))
			f, ok := factory.(func() (ptestPlugin, error))
			Expect(ok).To(BeTrue())
			plugin, err := f()
			Expect(err).To(Equal(ptestConfigurationFailedErr))
			Expect(plugin).To(BeNil())
		})

		It("no err, get config failed, throw panic", func() {
			factory := newFactoryOK(ptestNewConf, ptestNewType(), errToGetMaybe(ptestConfigurationFailedErr))
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			func() {
				defer func() {
					r := recover()
					Expect(r).To(Equal(ptestConfigurationFailedErr))
				}()
				f()
			}()
		})

		It("panic on trim non nil err", func() {
			factory := newFactoryOK(ptestNewErrFailing, ptestNewType(), nil)
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			func() {
				defer func() {
					r := recover()
					Expect(r).To(Equal(ptestCreateFailedErr))
				}()
				f()
			}()
		})

	})
})

var _ = Describe("factory constructor", func() {
	DescribeTable("expectations failed",
		func(newPlugin interface{}) {
			defer recoverExpectationFail()
			newFactoryConstructor(ptestType(), newPlugin)
		},
		Entry("not func",
			errors.New("that is not constructor")),
		Entry("returned not func",
			func() error { panic("") }),
		Entry("too many args",
			func(_, _ ptestConfig) func() ptestPlugin { panic("") }),
		Entry("too many return valued",
			func() (func() ptestPlugin, error, error) { panic("") }),
		Entry("second return value is not error",
			func() (func() ptestPlugin, ptestPlugin) { panic("") }),
		Entry("factory accepts conf",
			func() func(config ptestConfig) ptestPlugin { panic("") }),
		Entry("not implements",
			func() func() struct{} { panic("") }),
		Entry("factory too many args",
			func() func(_, _ ptestConfig) ptestPlugin { panic("") }),
		Entry("factory too many return valued",
			func() func() (_ ptestPlugin, _, _ error) { panic("") }),
		Entry("factory second return value is not error",
			func() func() (_, _ ptestPlugin) { panic("") }),
	)

	Context("new plugin", func() {
		newPlugin := func(newFactory interface{}, maybeConf []reflect.Value) (interface{}, error) {
			testee := newFactoryConstructor(ptestType(), newFactory)
			return testee.NewPlugin(maybeConf)
		}

		It("", func() {
			plugin, err := newPlugin(ptestNewFactory, nil)
			Expect(err).NotTo(HaveOccurred())
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("impl", func() {
			plugin, err := newPlugin(ptestNewFactoryImpl, nil)
			Expect(err).NotTo(HaveOccurred())
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("impl more than", func() {
			plugin, err := newPlugin(ptestNewFactoryMoreThan, nil)
			Expect(err).NotTo(HaveOccurred())
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("config", func() {
			plugin, err := newPlugin(ptestNewFactoryConf, confToMaybe(ptestDefaultConf()))
			Expect(err).NotTo(HaveOccurred())
			ptestExpectConfigValue(plugin, ptestDefaultValue)
		})

		It("failed", func() {
			plugin, err := newPlugin(ptestNewFactoryErrFailing, nil)
			Expect(err).To(Equal(ptestCreateFailedErr))
			Expect(plugin).To(BeNil())
		})

		It("factory failed", func() {
			plugin, err := newPlugin(ptestNewFactoryFactoryErrFailing, nil)
			Expect(err).To(Equal(ptestCreateFailedErr))
			Expect(plugin).To(BeNil())
		})
	})

	Context("new factory", func() {
		newFactory := func(newFactory interface{}, factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) (interface{}, error) {
			testee := newFactoryConstructor(ptestType(), newFactory)
			return testee.NewFactory(factoryType, getMaybeConf)
		}
		newFactoryOK := func(newF interface{}, factoryType reflect.Type, getMaybeConf func() ([]reflect.Value, error)) interface{} {
			factory, err := newFactory(newF, factoryType, getMaybeConf)
			Expect(err).NotTo(HaveOccurred())
			return factory
		}

		It("no err, same type - no wrap", func() {
			factory := newFactoryOK(ptestNewFactory, ptestNewType(), nil)
			expectSameFunc(factory, ptestNew)
		})

		It("has err, same type - no wrap", func() {
			factory := newFactoryOK(ptestNewFactoryFactoryErr, ptestNewErrType(), nil)
			expectSameFunc(factory, ptestNewErr)
		})

		It("from new impl", func() {
			factory := newFactoryOK(ptestNewFactoryImpl, ptestNewType(), nil)
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			plugin := f()
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("from new impl", func() {
			factory := newFactoryOK(ptestNewFactoryMoreThan, ptestNewType(), nil)
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			plugin := f()
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("add err", func() {
			factory := newFactoryOK(ptestNewFactory, ptestNewErrType(), nil)
			f, ok := factory.(func() (ptestPlugin, error))
			Expect(ok).To(BeTrue())
			plugin, err := f()
			Expect(err).NotTo(HaveOccurred())
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("factory construction not failed", func() {
			factory := newFactoryOK(ptestNewFactoryErr, ptestNewType(), nil)
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			plugin := f()
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("trim nil err", func() {
			factory := newFactoryOK(ptestNewFactoryFactoryErr, ptestNewType(), nil)
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			plugin := f()
			ptestExpectConfigValue(plugin, ptestInitValue)
		})

		It("config", func() {
			factory := newFactoryOK(ptestNewFactoryConf, ptestNewType(), confToGetMaybe(ptestDefaultConf()))
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			plugin := f()
			ptestExpectConfigValue(plugin, ptestDefaultValue)
		})

		It("get config failed", func() {
			factory, err := newFactory(ptestNewFactoryConf, ptestNewErrType(), errToGetMaybe(ptestConfigurationFailedErr))
			Expect(err).To(Equal(ptestConfigurationFailedErr))
			Expect(factory).To(BeNil())
		})

		It("factory create failed", func() {
			factory, err := newFactory(ptestNewFactoryErrFailing, ptestNewErrType(), nil)
			Expect(err).To(Equal(ptestCreateFailedErr))
			Expect(factory).To(BeNil())
		})

		It("plugin create failed", func() {
			factory := newFactoryOK(ptestNewFactoryFactoryErrFailing, ptestNewErrType(), nil)
			f, ok := factory.(func() (ptestPlugin, error))
			Expect(ok).To(BeTrue())
			plugin, err := f()
			Expect(err).To(Equal(ptestCreateFailedErr))
			Expect(plugin).To(BeNil())
		})

		It("panic on trim non nil err", func() {
			factory := newFactoryOK(ptestNewFactoryFactoryErrFailing, ptestNewType(), nil)
			f, ok := factory.(func() ptestPlugin)
			Expect(ok).To(BeTrue())
			func() {
				defer func() {
					r := recover()
					Expect(r).To(Equal(ptestCreateFailedErr))
				}()
				f()
			}()
		})

	})
})

func confToMaybe(conf interface{}) []reflect.Value {
	if conf != nil {
		return []reflect.Value{reflect.ValueOf(conf)}
	}
	return nil
}

func confToGetMaybe(conf interface{}) func() ([]reflect.Value, error) {
	return func() ([]reflect.Value, error) {
		return confToMaybe(conf), nil
	}
}

func errToGetMaybe(err error) func() ([]reflect.Value, error) {
	return func() ([]reflect.Value, error) {
		return nil, err
	}
}

func expectSameFunc(f1, f2 interface{}) {
	s1 := fmt.Sprint(f1)
	s2 := fmt.Sprint(f2)
	Expect(s1).To(Equal(s2))
}
