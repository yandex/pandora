package provider

import (
	"testing"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/ginkgoutil"
)

func TestProvider(t *testing.T) {
	ginkgoutil.RunSuite(t, "AmmoQueue Suite")
}

func testDeps() core.ProviderDeps {
	return core.ProviderDeps{Log: ginkgoutil.NewLogger()}
}
