package plugin

import (
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

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
func testPluginImplFactoryType() reflect.Type {
	return reflect.TypeOf(func() (*testPluginImpl, error) { panic("") })
}
func testPluginNoErrFactoryType() reflect.Type {
	return reflect.TypeOf(func() testPlugin { panic("") })
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

func newTestPluginDefaultConf() testPluginConfig     { return testPluginConfig{testDefaultValue} }
func newTestPluginDefaultPtrConf() *testPluginConfig { return &testPluginConfig{testDefaultValue} }

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
