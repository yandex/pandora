// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"reflect"

	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var _ = Describe("plugin constructor", func() {
	DescribeTable("expectations failed",
		func(pluginType reflect.Type, newPlugin interface{}) {
			defer recoverExpectationFail()
			newPluginConstructor(pluginType, newPlugin)
			Expect(true)
		},
		Entry("not func",
			errorType, errors.New("that is not constructor")),
		Entry("not implements",
			errorType, ptestNewImpl),
		Entry("too many args",
			ptestType(), func(_, _ ptestConfig) ptestPlugin { panic("") }),
		Entry("too many return valued",
			ptestType(), func() (_ ptestPlugin, _, _ error) { panic("") }),
		Entry("second return value is not error",
			ptestType(), func() (_, _ ptestPlugin) { panic("") }),
	)
	confToMaybe := func(conf interface{}) []reflect.Value {
		if conf != nil {
			return []reflect.Value{reflect.ValueOf(conf)}
		}
		return nil
	}

	confToGetMaybe := func(conf interface{}) func() ([]reflect.Value, error) {
		return func() ([]reflect.Value, error) {
			return confToMaybe(conf), nil
		}
	}

	errToGetMaybe := func(err error) func() ([]reflect.Value, error) {
		return func() ([]reflect.Value, error) {
			return nil, err
		}
	}

	It("new plugin", func() {
		testee := newPluginConstructor(ptestType(), ptestNew)
		plugin, err := testee.NewPlugin(nil)
		Expect(err).NotTo(HaveOccurred())
		ptestExpectConfigValue(plugin, ptestInitValue)
	})

	It("new config plugin ", func() {
		testee := newPluginConstructor(ptestType(), ptestNewImplConf)
		plugin, err := testee.NewPlugin(confToMaybe(ptestDefaultConf()))
		Expect(err).NotTo(HaveOccurred())
		ptestExpectConfigValue(plugin, ptestDefaultValue)
	})

	It("new plugin failed", func() {
		testee := newPluginConstructor(ptestType(), ptestNewImplErrFailed)
		plugin, err := testee.NewPlugin(nil)
		Expect(err).To(Equal(ptestCreateFailedErr))
		Expect(plugin).To(BeNil())
	})

	It("new factory from conversion", func() {
		testee := newPluginConstructor(ptestType(), ptestNew)
		factory, err := testee.NewFactory(ptestFactoryNoErrType(), nil)
		Expect(err).NotTo(HaveOccurred())
		expectSameFunc(factory, ptestNew)
	})

	It("new factory from new impl", func() {
		testee := newPluginConstructor(ptestType(), ptestNewImpl)
		factory, err := testee.NewFactory(ptestFactoryNoErrType(), nil)
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() ptestPlugin)
		Expect(ok).To(BeTrue())
		plugin := f()
		ptestExpectConfigValue(plugin, ptestInitValue)
	})

	It("new factory add err", func() {
		testee := newPluginConstructor(ptestType(), ptestNew)
		factory, err := testee.NewFactory(ptestFactoryType(), nil)
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() (ptestPlugin, error))
		Expect(ok).To(BeTrue())
		plugin, err := f()
		Expect(err).NotTo(HaveOccurred())
		ptestExpectConfigValue(plugin, ptestInitValue)
	})

	It("new factory trim nil err", func() {
		testee := newPluginConstructor(ptestType(), ptestNewImplErr)
		factory, err := testee.NewFactory(ptestFactoryNoErrType(), nil)
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() ptestPlugin)
		Expect(ok).To(BeTrue())
		plugin := f()
		ptestExpectConfigValue(plugin, ptestInitValue)
	})

	It("new factory config", func() {
		testee := newPluginConstructor(ptestType(), ptestNewImplConf)
		factory, err := testee.NewFactory(ptestFactoryNoErrType(), confToGetMaybe(ptestDefaultConf()))
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() ptestPlugin)
		Expect(ok).To(BeTrue())
		plugin := f()
		ptestExpectConfigValue(plugin, ptestDefaultValue)
	})

	It("new factory, get config failed", func() {
		testee := newPluginConstructor(ptestType(), ptestNewImplConf)
		factory, err := testee.NewFactory(ptestFactoryType(), errToGetMaybe(ptestConfigurationFailedErr))
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() (ptestPlugin, error))
		Expect(ok).To(BeTrue())
		plugin, err := f()
		Expect(err).To(Equal(ptestConfigurationFailedErr))
		Expect(plugin).To(BeNil())
	})

	It("new factory no err, get config failed, throw panic", func() {
		testee := newPluginConstructor(ptestType(), ptestNewImplConf)
		factory, err := testee.NewFactory(ptestFactoryNoErrType(), errToGetMaybe(ptestConfigurationFailedErr))
		Expect(err).NotTo(HaveOccurred())
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

	It("new factory panic on trim non nil err", func() {
		testee := newPluginConstructor(ptestType(), ptestNewImplErrFailed)
		factory, err := testee.NewFactory(ptestFactoryNoErrType(), nil)
		Expect(err).NotTo(HaveOccurred())
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

func expectSameFunc(f1, f2 interface{}) {
	s1 := fmt.Sprint(f1)
	s2 := fmt.Sprint(f2)
	Expect(s1).To(Equal(s2))
}
