// Copyright (c) 2017 Yandex LLC. All rights reserved.
// Use of this source code is governed by a MPL 2.0
// license that can be found in the LICENSE file.
// Author: Vladimir Skipor <skipor@yandex-team.ru>

package plugin

import (
	"reflect"

	. "github.com/onsi/gomega"
	"github.com/pkg/errors"

	"a.yandex-team.ru/load/projects/pandora/core/config"
)

// ptest contains examples and utils for testing plugin pkg

const (
	ptestPluginName = "ptest_name"

	ptestInitValue    = "ptest_INITIAL"
	ptestDefaultValue = "ptest_DEFAULT_CONFIG"
	ptestFilledValue  = "ptest_FILLED"
)

func (r *Registry) ptestRegister(constructor interface{}, newDefaultConfigOptional ...interface{}) {
	r.Register(ptestType(), ptestPluginName, constructor, newDefaultConfigOptional...)
}
func (r *Registry) ptestNew(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	return r.New(ptestType(), ptestPluginName, fillConfOptional...)
}
func (r *Registry) ptestNewFactory(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	factory, err := r.NewFactory(ptestNewErrType(), ptestPluginName, fillConfOptional...)
	if err != nil {
		return
	}
	typedFactory := factory.(func() (ptestPlugin, error))
	return typedFactory()
}

var (
	ptestCreateFailedErr        = errors.New("test plugin create failed")
	ptestConfigurationFailedErr = errors.New("test plugin configuration failed")
)

type ptestPlugin interface {
	DoSomething()
}
type ptestMoreThanPlugin interface {
	ptestPlugin
	DoSomethingElse()
}
type ptestImpl struct{ Value string }
type ptestConfig struct{ Value string }

func (p *ptestImpl) DoSomething()     {}
func (p *ptestImpl) DoSomethingElse() {}

func ptestNew() ptestPlugin                      { return ptestNewImpl() }
func ptestNewMoreThan() ptestMoreThanPlugin      { return ptestNewImpl() }
func ptestNewImpl() *ptestImpl                   { return &ptestImpl{Value: ptestInitValue} }
func ptestNewConf(c ptestConfig) ptestPlugin     { return &ptestImpl{c.Value} }
func ptestNewPtrConf(c *ptestConfig) ptestPlugin { return &ptestImpl{c.Value} }
func ptestNewErr() (ptestPlugin, error)          { return &ptestImpl{Value: ptestInitValue}, nil }
func ptestNewErrFailing() (ptestPlugin, error)   { return nil, ptestCreateFailedErr }

func ptestNewFactory() func() ptestPlugin                 { return ptestNew }
func ptestNewFactoryMoreThan() func() ptestMoreThanPlugin { return ptestNewMoreThan }
func ptestNewFactoryImpl() func() *ptestImpl              { return ptestNewImpl }
func ptestNewFactoryConf(c ptestConfig) func() ptestPlugin {
	return func() ptestPlugin {
		return ptestNewConf(c)
	}
}
func ptestNewFactoryPtrConf(c *ptestConfig) func() ptestPlugin {
	return func() ptestPlugin {
		return ptestNewPtrConf(c)
	}
}
func ptestNewFactoryErr() (func() ptestPlugin, error)               { return ptestNew, nil }
func ptestNewFactoryErrFailing() (func() ptestPlugin, error)        { return nil, ptestCreateFailedErr }
func ptestNewFactoryFactoryErr() func() (ptestPlugin, error)        { return ptestNewErr }
func ptestNewFactoryFactoryErrFailing() func() (ptestPlugin, error) { return ptestNewErrFailing }

func ptestDefaultConf() ptestConfig        { return ptestConfig{ptestDefaultValue} }
func ptestNewDefaultPtrConf() *ptestConfig { return &ptestConfig{ptestDefaultValue} }

func ptestType() reflect.Type       { return PtrType((*ptestPlugin)(nil)) }
func ptestNewErrType() reflect.Type { return reflect.TypeOf(ptestNewErr) }
func ptestNewType() reflect.Type    { return reflect.TypeOf(ptestNew) }

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
