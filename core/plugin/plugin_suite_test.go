// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pkg/errors"
	"github.com/yandex/pandora/core/config"
	"github.com/yandex/pandora/lib/testutil"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	testutil.ReplaceGlobalLogger()
	RunSpecs(t, "Plugin Suite")
}

const (
	testPluginName   = "test_name"
	testDefaultValue = "default"
	testInitValue    = "init"
	testFilledValue  = "conf"
)

func (r *Registry) testRegister(newPluginImpl interface{}, newDefaultConfigOptional ...interface{}) {
	r.Register(testPluginType(), testPluginName, newPluginImpl, newDefaultConfigOptional...)
}

func (r *Registry) testNew(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	return r.New(testPluginType(), testPluginName, fillConfOptional...)
}

func (r *Registry) testNewFactory(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	factory, err := r.NewFactory(testPluginFactoryType(), testPluginName, fillConfOptional...)
	if err != nil {
		return
	}
	typedFactory := factory.(func() (testPlugin, error))
	return typedFactory()
}

type testPlugin interface {
	DoSomething()
}

func testPluginType() reflect.Type     { return reflect.TypeOf((*testPlugin)(nil)).Elem() }
func testPluginImplType() reflect.Type { return reflect.TypeOf((*testPluginImpl)(nil)).Elem() }

func testPluginFactoryType() reflect.Type {
	return reflect.TypeOf(func() (testPlugin, error) { panic("") })
}
func testPluginNoErrFactoryType() reflect.Type {
	return reflect.TypeOf(func() testPlugin { panic("") })
}
func testPluginFactoryConfigType() reflect.Type {
	return reflect.TypeOf(func(testPluginConfig) (testPlugin, error) { panic("") })
}

type testPluginImpl struct{ Value string }

func (p *testPluginImpl) DoSomething() {}

var _ testPlugin = (*testPluginImpl)(nil)

type testPluginConfig struct{ Value string }

func newTestPlugin() testPlugin                                { return newTestPluginImpl() }
func newTestPluginImpl() *testPluginImpl                       { return &testPluginImpl{Value: testInitValue} }
func newTestPluginImplConf(c testPluginConfig) *testPluginImpl { return &testPluginImpl{c.Value} }
func newTestPluginImplPtrConf(c *testPluginConfig) *testPluginImpl {
	return &testPluginImpl{c.Value}
}

func newTestPluginImplErr() (*testPluginImpl, error) {
	return &testPluginImpl{Value: testInitValue}, nil
}

var testPluginCreateFailedErr = errors.New("test plugin create failed")
var testConfigurationFailedErr = errors.New("test plugin configuration failed")

func newTestPluginImplErrFailed() (*testPluginImpl, error) {
	return nil, testPluginCreateFailedErr
}

func newTestDefaultConf() testPluginConfig     { return testPluginConfig{testDefaultValue} }
func newTestDefaultPtrConf() *testPluginConfig { return &testPluginConfig{testDefaultValue} }

func fillTestPluginConf(conf interface{}) error {
	return config.Decode(map[string]interface{}{"Value": testFilledValue}, conf)
}

func expectConfigValue(conf interface{}, val string) {
	conf.(confChecker).expectValue(val)
}

type confChecker interface {
	expectValue(string)
}

var _ confChecker = testPluginConfig{}
var _ confChecker = &testPluginImpl{}

func (c testPluginConfig) expectValue(val string) { Expect(c.Value).To(Equal(val)) }
func (p *testPluginImpl) expectValue(val string)  { Expect(p.Value).To(Equal(val)) }
