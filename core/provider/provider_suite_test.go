package provider

import (
	"testing"

	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/lib/testutil"
)

func TestProvider(t *testing.T) {
	testutil.RunSuite(t, "AmmoQueue Suite")
}

func testDeps() core.ProviderDeps {
	return core.ProviderDeps{testutil.NewLogger()}
}
