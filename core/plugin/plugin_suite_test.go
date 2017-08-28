package plugin

import (
	"reflect"
	"testing"

	"github.com/mitchellh/mapstructure"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/yandex/pandora/lib/testutil"
)

func TestPlugin(t *testing.T) {
	RegisterFailHandler(Fail)
	testutil.ReplaceGlobalLogger()
	RunSpecs(t, "Plugin Suite")
}

const (
	testPluginName   = "test_name"
	testConfValue    = "conf"
	testDefaultValue = "default"
	testInitValue    = "init"
)

func (r typeRegistry) testRegister(newPluginImpl interface{}, newDefaultConfigOptional ...interface{}) {
	r.Register(testPluginType(), testPluginName, newPluginImpl, newDefaultConfigOptional...)
}

func (r typeRegistry) testNew(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	return r.New(testPluginType(), testPluginName, fillConfOptional...)
}

func (r typeRegistry) testNewFactory(fillConfOptional ...func(conf interface{}) error) (plugin interface{}, err error) {
	factory, err := r.NewFactory(testPluginFactoryType(), testPluginName, fillConfOptional...)
	if err != nil {
		return
	}
	typedFactory := factory.(func() (testPluginInterface, error))
	return typedFactory()
}

type testPluginInterface interface {
	DoSomething()
}

func testPluginType() reflect.Type { return reflect.TypeOf((*testPluginInterface)(nil)).Elem() }
func testPluginFactoryType() reflect.Type {
	return reflect.TypeOf(func() (testPluginInterface, error) { panic("") })
}
func newTestPlugin() *testPluginImpl { return &testPluginImpl{Value: testInitValue} }

type testPluginImpl struct{ Value string }

func (p *testPluginImpl) DoSomething() {}

var _ testPluginInterface = (*testPluginImpl)(nil)

type testPluginImplConfig struct{ Value string }

func newTestPluginConf(c testPluginImplConfig) *testPluginImpl { return &testPluginImpl{c.Value} }
func newTestPluginDefaultConf() testPluginImplConfig           { return testPluginImplConfig{testDefaultValue} }
func newTestPluginPtrConf(c *testPluginImplConfig) *testPluginImpl {
	return &testPluginImpl{c.Value}
}

func fillTestPluginConf(conf interface{}) error {
	return mapstructure.Decode(map[string]interface{}{"Value": "conf"}, conf)
}
