package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultRegistry(t *testing.T) {
	beforeEach := func() { Register(ptestType(), ptestPluginName, ptestNewImpl) }
	afterEach := func() { defaultRegistry = NewRegistry() }

	t.Run("lookup", func(t *testing.T) {
		defer afterEach()
		beforeEach()
		assert.True(t, Lookup(ptestType()))
	})
	t.Run("lookup factory", func(t *testing.T) {
		defer afterEach()
		beforeEach()
		assert.True(t, LookupFactory(ptestNewErrType()))
	})
	t.Run("new", func(t *testing.T) {
		defer afterEach()
		beforeEach()
		plugin, err := New(ptestType(), ptestPluginName)
		assert.NoError(t, err)
		assert.NotNil(t, plugin)
	})
	t.Run("new factory", func(t *testing.T) {
		defer afterEach()
		beforeEach()
		pluginFactory, err := NewFactory(ptestNewErrType(), ptestPluginName)
		assert.NoError(t, err)
		assert.NotNil(t, pluginFactory)
	})
}

func TestTypeHelpers(t *testing.T) {
	t.Run("ptr type", func(t *testing.T) {
		var plugin ptestPlugin
		assert.Equal(t, ptestType(), PtrType(&plugin))
	})
	t.Run("factory plugin type ok", func(t *testing.T) {
		factoryPlugin, ok := FactoryPluginType(ptestNewErrType())
		assert.True(t, ok)
		assert.Equal(t, ptestType(), factoryPlugin)
	})
}
