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
			errorType, newTestPluginImpl),
		Entry("too many args",
			testPluginType(), func(_, _ testPluginConfig) testPlugin { panic("") }),
		Entry("too many return valued",
			testPluginType(), func() (_ testPlugin, _, _ error) { panic("") }),
		Entry("second return value is not error",
			testPluginType(), func() (_, _ testPlugin) { panic("") }),
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
		testee := newPluginConstructor(testPluginType(), newTestPlugin)
		plugin, err := testee.NewPlugin(nil)
		Expect(err).NotTo(HaveOccurred())
		expectConfigValue(plugin, testInitValue)
	})

	It("new config plugin ", func() {
		testee := newPluginConstructor(testPluginType(), newTestPluginImplConf)
		plugin, err := testee.NewPlugin(confToMaybe(newTestDefaultConf()))
		Expect(err).NotTo(HaveOccurred())
		expectConfigValue(plugin, testDefaultValue)
	})

	It("new plugin failed", func() {
		testee := newPluginConstructor(testPluginType(), newTestPluginImplErrFailed)
		plugin, err := testee.NewPlugin(nil)
		Expect(err).To(Equal(testPluginCreateFailedErr))
		Expect(plugin).To(BeNil())
	})

	It("new factory from conversion", func() {
		testee := newPluginConstructor(testPluginType(), newTestPlugin)
		factory, err := testee.NewFactory(testPluginNoErrFactoryType(), nil)
		Expect(err).NotTo(HaveOccurred())
		expectSameFunc(factory, newTestPlugin)
	})

	It("new factory from new impl", func() {
		testee := newPluginConstructor(testPluginType(), newTestPluginImpl)
		factory, err := testee.NewFactory(testPluginNoErrFactoryType(), nil)
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() testPlugin)
		Expect(ok).To(BeTrue())
		plugin := f()
		expectConfigValue(plugin, testInitValue)
	})

	It("new factory add err", func() {
		testee := newPluginConstructor(testPluginType(), newTestPlugin)
		factory, err := testee.NewFactory(testPluginFactoryType(), nil)
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() (testPlugin, error))
		Expect(ok).To(BeTrue())
		plugin, err := f()
		Expect(err).NotTo(HaveOccurred())
		expectConfigValue(plugin, testInitValue)
	})

	It("new factory trim nil err", func() {
		testee := newPluginConstructor(testPluginType(), newTestPluginImplErr)
		factory, err := testee.NewFactory(testPluginNoErrFactoryType(), nil)
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() testPlugin)
		Expect(ok).To(BeTrue())
		plugin := f()
		expectConfigValue(plugin, testInitValue)
	})

	It("new factory config", func() {
		testee := newPluginConstructor(testPluginType(), newTestPluginImplConf)
		factory, err := testee.NewFactory(testPluginNoErrFactoryType(), confToGetMaybe(newTestDefaultConf()))
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() testPlugin)
		Expect(ok).To(BeTrue())
		plugin := f()
		expectConfigValue(plugin, testDefaultValue)
	})

	It("new factory, get config failed", func() {
		testee := newPluginConstructor(testPluginType(), newTestPluginImplConf)
		factory, err := testee.NewFactory(testPluginFactoryType(), errToGetMaybe(testConfigurationFailedErr))
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() (testPlugin, error))
		Expect(ok).To(BeTrue())
		plugin, err := f()
		Expect(err).To(Equal(testConfigurationFailedErr))
		Expect(plugin).To(BeNil())
	})

	It("new factory no err, get config failed, throw panic", func() {
		testee := newPluginConstructor(testPluginType(), newTestPluginImplConf)
		factory, err := testee.NewFactory(testPluginNoErrFactoryType(), errToGetMaybe(testConfigurationFailedErr))
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() testPlugin)
		Expect(ok).To(BeTrue())
		func() {
			defer func() {
				r := recover()
				Expect(r).To(Equal(testConfigurationFailedErr))
			}()
			f()
		}()
	})

	It("new factory panic on trim non nil err", func() {
		testee := newPluginConstructor(testPluginType(), newTestPluginImplErrFailed)
		factory, err := testee.NewFactory(testPluginNoErrFactoryType(), nil)
		Expect(err).NotTo(HaveOccurred())
		f, ok := factory.(func() testPlugin)
		Expect(ok).To(BeTrue())
		func() {
			defer func() {
				r := recover()
				Expect(r).To(Equal(testPluginCreateFailedErr))
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
