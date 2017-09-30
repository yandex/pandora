// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"reflect"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"github.com/yandex/pandora/core/config"
)

// ptest contains samples and utils for testing plugin pkg

const (
	ptestPluginName = "ptest_name"

	ptestInitValue    = "ptest_INITIAL"
	ptestDefaultValue = "ptest_DEFAULT_CONFIG"
	ptestFilledValue  = "ptest_FILLED"
)

func (r *Registry) ptestRegister(newPluginImpl interface{}, newDefaultConfigOptional ...interface{}) {
	r.Register(ptestType(), ptestPluginName, newPluginImpl, newDefaultConfigOptional...)
}

func (r *Registry) ptestNew(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	return r.New(ptestType(), ptestPluginName, fillConfOptional...)
}

func (r *Registry) ptestNewFactory(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	factory, err := r.NewFactory(ptestFactoryType(), ptestPluginName, fillConfOptional...)
	if err != nil {
		return
	}
	typedFactory := factory.(func() (ptestPlugin, error))
	return typedFactory()
}

type ptestPlugin interface {
	DoSomething()
}

func ptestType() reflect.Type     { return reflect.TypeOf((*ptestPlugin)(nil)).Elem() }
func ptestImplType() reflect.Type { return reflect.TypeOf((*ptestImpl)(nil)).Elem() }

func ptestFactoryType() reflect.Type {
	return reflect.TypeOf(func() (ptestPlugin, error) { panic("") })
}
func ptestFactoryNoErrType() reflect.Type {
	return reflect.TypeOf(func() ptestPlugin { panic("") })
}
func ptestFactoryConfigType() reflect.Type {
	return reflect.TypeOf(func(ptestConfig) (ptestPlugin, error) { panic("") })
}

type ptestImpl struct{ Value string }

func (p *ptestImpl) DoSomething() {}

var _ ptestPlugin = (*ptestImpl)(nil)

type ptestConfig struct{ Value string }

func ptestNew() ptestPlugin                     { return ptestNewImpl() }
func ptestNewImpl() *ptestImpl                  { return &ptestImpl{Value: ptestInitValue} }
func ptestNewImplConf(c ptestConfig) *ptestImpl { return &ptestImpl{c.Value} }
func ptestNewImplPtrConf(c *ptestConfig) *ptestImpl {
	return &ptestImpl{c.Value}
}

func ptestNewImplErr() (*ptestImpl, error) {
	return &ptestImpl{Value: ptestInitValue}, nil
}

var ptestCreateFailedErr = errors.New("test plugin create failed")
var ptestConfigurationFailedErr = errors.New("test plugin configuration failed")

func ptestNewImplErrFailed() (*ptestImpl, error) {
	return nil, ptestCreateFailedErr
}

func ptestDefaultConf() ptestConfig        { return ptestConfig{ptestDefaultValue} }
func ptestNewDefaultPtrConf() *ptestConfig { return &ptestConfig{ptestDefaultValue} }

func ptestFillConf(conf interface{}) error {
	return config.Decode(map[string]interface{}{"Value": ptestFilledValue}, conf)
}

func ptestExpectConfigValue(conf interface{}, val string) {
	conf.(ptestConfChecker).expectConfValue(val)
}

type ptestConfChecker interface {
	expectConfValue(string)
}

var _ ptestConfChecker = ptestConfig{}
var _ ptestConfChecker = &ptestImpl{}

func (c ptestConfig) expectConfValue(val string) { Expect(c.Value).To(Equal(val)) }
func (p *ptestImpl) expectConfValue(val string)  { Expect(p.Value).To(Equal(val)) }
