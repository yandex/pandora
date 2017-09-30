// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"reflect"

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
	var (
		conf interface{}
		// Auto set.
		//getMaybeConf = func() []reflect.Value {
		//	return confToMaybe(conf)
		//}
	)
	BeforeEach(func() { conf = nil })

	It("new plugin", func() {
		testee := newPluginConstructor(testPluginType(), newTestPlugin)
		plugin, err := testee.NewPlugin(nil)
		Expect(err).NotTo(HaveOccurred())
		expectConfigValue(plugin, testInitValue)
	})

	It("new config plugin ", func() {
		testee := newPluginConstructor(testPluginType(), newTestPluginImplConf)
		plugin, err := testee.NewPlugin(confToMaybe(newTestPluginDefaultConf()))
		Expect(err).NotTo(HaveOccurred())
		expectConfigValue(plugin, testDefaultValue)
	})

})
